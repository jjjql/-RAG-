// Package main：网关进程入口：加载配置、启动 HTTP、优雅退出。
package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"rag-gateway/internal/bootstrap"
	"rag-gateway/internal/circuitbreaker"
	"rag-gateway/internal/coalesce"
	"rag-gateway/internal/config"
	"rag-gateway/internal/downstream"
	"rag-gateway/internal/embedding"
	httpnb "rag-gateway/internal/northbound/http"
	_ "rag-gateway/internal/observability" // Prometheus 指标注册（/metrics）
	"rag-gateway/internal/rules"
	"rag-gateway/internal/vector"
)

func main() {
	configPath := flag.String("config", "config.yaml", "配置文件路径（YAML）")
	shutdownGrace := flag.Duration("shutdown-grace", 15*time.Second, "收到退出信号后优雅关闭最长等待时间")
	defaultAddr := flag.String("http-addr-default", ":8080", "当配置未写 server.http_addr 时使用的默认监听地址")
	flag.Parse()

	cfg, err := config.Load(*configPath, *defaultAddr)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Redis.Addr == "" {
		log.Fatal("配置 redis.addr 不能为空（SYS-FUNC-01 依赖 Redis）")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = rdb.Ping(ctx).Err()
	cancel()
	if err != nil {
		log.Fatalf("连接 Redis 失败: %v", err)
	}

	store := rules.NewRedisExactStore(rdb)
	mem := rules.NewExactMemoryIndex()
	coord := rules.NewExactCoordinator(store, mem)
	if err := coord.Reload(context.Background()); err != nil {
		log.Fatalf("加载精确规则到内存失败: %v", err)
	}

	subCtx, subCancel := context.WithCancel(context.Background())
	defer subCancel()
	go subscribeExactReload(subCtx, rdb, coord)

	regexStore := rules.NewRedisRegexStore(rdb)
	regexMem := rules.NewRegexMemoryIndex()
	regexCoord := rules.NewRegexCoordinator(regexStore, regexMem)
	if err := regexCoord.Reload(context.Background()); err != nil {
		log.Fatalf("加载正则规则到内存失败: %v", err)
	}
	go subscribeRegexReload(subCtx, rdb, regexCoord)

	var embedSvc embedding.Service
	if cfg.Embedding.Enabled {
		ec := embedding.ClientConfig{
			Transport:  cfg.Embedding.Transport,
			SocketPath: cfg.Embedding.SocketPath,
			TCPAddr:    cfg.Embedding.TCPAddr,
		}
		if cfg.Embedding.TimeoutMS > 0 {
			ec.Timeout = time.Duration(cfg.Embedding.TimeoutMS) * time.Millisecond
		}
		if cfg.Embedding.PingTimeoutMS > 0 {
			ec.PingTimeout = time.Duration(cfg.Embedding.PingTimeoutMS) * time.Millisecond
		}
		if ec.Transport == "" {
			ec.Transport = "unix"
		}
		if ec.Transport == "unix" && ec.SocketPath == "" {
			ec.SocketPath = "/tmp/rag_gateway.sock"
		}
		if ec.Transport == "tcp" && ec.TCPAddr == "" {
			log.Fatal("embedding.enabled=true 且 transport=tcp 时必须配置 embedding.tcp_addr")
		}
		emb := embedding.NewClient(ec)
		defer func() { _ = emb.Close() }()
		pctx, pcancel := context.WithTimeout(context.Background(), ec.PingTimeout)
		if err := emb.Ping(pctx); err != nil {
			log.Printf("警告: 侧车 Embedding ping 失败（确认 ai_service 已就绪）: %v", err)
		}
		pcancel()
		embedSvc = emb
		if cfg.Embedding.CircuitBreaker.Enabled {
			embedSvc = &embedding.CircuitService{
				Inner:   emb,
				Breaker: newBreakerFromCfg(cfg.Embedding.CircuitBreaker),
			}
		}
	}

	var rag *downstream.Client
	if cfg.Downstream.Enabled {
		mode := cfg.Downstream.Mode
		if mode == "" {
			mode = "mock"
		}
		to := time.Duration(cfg.Downstream.TimeoutMS) * time.Millisecond
		var ans downstream.Answerer
		switch mode {
		case "mock":
			m := downstream.NewMock(cfg.Downstream.MockText)
			if d := cfg.Downstream.MockDelayMS; d > 0 {
				m.Delay = time.Duration(d) * time.Millisecond
				log.Printf("智能问答下游 mock（timeout=%v，mock_delay_ms=%d 用于故障注入）", to, d)
			} else {
				log.Printf("智能问答下游 mock（timeout=%v）", to)
			}
			ans = m
		case "langchain_http":
			base := strings.TrimSpace(cfg.Downstream.HTTPBaseURL)
			if base == "" {
				log.Fatal("downstream.mode=langchain_http 须配置 downstream.http_base_url")
			}
			p := strings.TrimSpace(cfg.Downstream.HTTPPath)
			if p == "" {
				p = "/v1/rag/invoke"
			}
			if !strings.HasPrefix(p, "/") {
				log.Fatal("downstream.http_path 须以 / 开头")
			}
			ans = downstream.NewLangChainHTTP(downstream.LangChainHTTPConfig{
				BaseURL:      base,
				Path:         p,
				APIKeyHeader: strings.TrimSpace(cfg.Downstream.HTTPAPIKeyHeader),
				APIKey:       strings.TrimSpace(cfg.Downstream.HTTPAPIKey),
			})
			log.Printf("智能问答下游 LangChain HTTP %s%s（timeout=%v）", base, p, to)
		default:
			log.Fatalf("downstream.mode=%q 不支持（支持 mock、langchain_http）", mode)
		}
		if cfg.Downstream.CircuitBreaker.Enabled {
			ans = downstream.WrapAnswerer(ans, newBreakerFromCfg(cfg.Downstream.CircuitBreaker))
		}
		rag = &downstream.Client{
			A:       ans,
			Timeout: to,
		}
	}

	var vecStore vector.Store = vector.Noop{}
	var dedupSearch vector.Store
	var dedupWriter vector.AnswerWriter
	vectorEnabled := cfg.Vector.Enabled
	if cfg.Vector.Enabled {
		vm := strings.TrimSpace(strings.ToLower(cfg.Vector.Mode))
		if vm == "" {
			vm = "noop"
		}
		vto := time.Duration(cfg.Vector.TimeoutMS) * time.Millisecond
		if vto <= 0 {
			vto = 100 * time.Millisecond
		}
		switch vm {
		case "qdrant":
			base := strings.TrimSpace(cfg.Vector.Qdrant.URL)
			coll := strings.TrimSpace(cfg.Vector.Qdrant.Collection)
			if base == "" || coll == "" {
				log.Fatal("vector.enabled=true 且 mode=qdrant 须配置 vector.qdrant.url 与 vector.qdrant.collection")
			}
			th := cfg.Vector.ScoreThreshold
			q := vector.NewQdrant(vector.QdrantConfig{
				BaseURL:        base,
				Collection:     coll,
				APIKey:         strings.TrimSpace(cfg.Vector.Qdrant.APIKey),
				ScoreThreshold: th,
				HTTPClient:     &http.Client{Timeout: vto},
			})
			var vbr *circuitbreaker.Breaker
			if cfg.Vector.CircuitBreaker.Enabled {
				vbr = newBreakerFromCfg(cfg.Vector.CircuitBreaker)
			}
			vecStore = vector.NewCircuitStore(q, vbr, vto)
			log.Printf("向量检索 Qdrant（timeout=%v，熔断=%v）", vto, cfg.Vector.CircuitBreaker.Enabled)
		default:
			vecStore = vector.Noop{}
			log.Printf("向量 mode=%q 使用 noop", vm)
		}
	}

	if cfg.Vector.SemanticDedup.Enabled {
		if !cfg.Embedding.Enabled {
			log.Fatal("vector.semantic_dedup.enabled=true 须同时 embedding.enabled=true")
		}
		vm := strings.TrimSpace(strings.ToLower(cfg.Vector.Mode))
		if !cfg.Vector.Enabled || vm != "qdrant" {
			log.Fatal("vector.semantic_dedup 仅支持 vector.enabled=true 且 mode=qdrant")
		}
		vto := time.Duration(cfg.Vector.TimeoutMS) * time.Millisecond
		if vto <= 0 {
			vto = 100 * time.Millisecond
		}
		base := strings.TrimSpace(cfg.Vector.Qdrant.URL)
		primaryColl := strings.TrimSpace(cfg.Vector.Qdrant.Collection)
		if base == "" || primaryColl == "" {
			log.Fatal("semantic_dedup 需要有效的 vector.qdrant.url 与 collection")
		}
		dedupColl := strings.TrimSpace(cfg.Vector.SemanticDedup.Collection)
		dedupTh := cfg.Vector.SemanticDedup.ScoreThreshold
		if dedupTh <= 0 {
			dedupTh = cfg.Vector.ScoreThreshold
		}
		var vbr *circuitbreaker.Breaker
		if cfg.Vector.CircuitBreaker.Enabled {
			vbr = newBreakerFromCfg(cfg.Vector.CircuitBreaker)
		}
		writeColl := primaryColl
		if dedupColl != "" {
			writeColl = dedupColl
		}
		qWrite := vector.NewQdrant(vector.QdrantConfig{
			BaseURL:        base,
			Collection:     writeColl,
			APIKey:         strings.TrimSpace(cfg.Vector.Qdrant.APIKey),
			ScoreThreshold: dedupTh,
			HTTPClient:     &http.Client{Timeout: vto},
		})
		dedupWriter = &vector.TimeoutUpserter{Inner: qWrite, Timeout: 3 * time.Second}
		log.Printf("持久化语义去重：写回 collection=%s（wait=true upsert）", writeColl)
		if dedupColl != "" && !strings.EqualFold(dedupColl, primaryColl) {
			qDedup := vector.NewQdrant(vector.QdrantConfig{
				BaseURL:        base,
				Collection:     dedupColl,
				APIKey:         strings.TrimSpace(cfg.Vector.Qdrant.APIKey),
				ScoreThreshold: dedupTh,
				HTTPClient:     &http.Client{Timeout: vto},
			})
			dedupSearch = vector.NewCircuitStore(qDedup, vbr, vto)
			log.Printf("持久化语义去重：独立检索集合 %s（阈值=%v）", dedupColl, dedupTh)
		}
	}

	var sem coalesce.Semantic
	merger := coalesce.Merger(coalesce.Passthrough{})
	if cfg.Coalesce.Enabled {
		mergeTO := time.Duration(cfg.Downstream.TimeoutMS) * time.Millisecond
		if mergeTO <= 0 {
			mergeTO = 100 * time.Millisecond
		}
		mode := strings.TrimSpace(strings.ToLower(cfg.Coalesce.Mode))
		if mode == "" {
			mode = "local"
		}
		var redisCfg coalesce.RedisConfig
		switch mode {
		case "redis":
			lockTTL := time.Duration(cfg.Coalesce.LockTTLSeconds) * time.Second
			if lockTTL <= 0 {
				lockTTL = 120 * time.Second
			}
			resTTL := time.Duration(cfg.Coalesce.ResultTTLSeconds) * time.Second
			if resTTL <= 0 {
				resTTL = 90 * time.Second
			}
			redisCfg = coalesce.RedisConfig{
				MergeTimeout: mergeTO,
				LockTTL:      lockTTL,
				ResultTTL:    resTTL,
			}
			merger = coalesce.NewRedis(rdb, redisCfg)
			log.Printf("相似请求合并 coalesce=redis（MergeTimeout=%v，lock=%v，resultTTL=%v）", mergeTO, lockTTL, resTTL)
		default:
			merger = &coalesce.RAG{Enabled: true, MergeTimeout: mergeTO}
			log.Printf("相似请求合并 coalesce=local（MergeTimeout=%v，键=scope+规范化query）", mergeTO)
		}
		if cfg.Coalesce.Semantic {
			if embedSvc == nil {
				log.Fatal("coalesce.semantic=true 时必须启用 embedding.enabled 并提供侧车")
			}
			th := cfg.Coalesce.SimilarityThreshold
			if th <= 0 {
				th = 0.95
			}
			maxA := cfg.Coalesce.SemanticMaxActive
			if maxA <= 0 {
				maxA = 256
			}
			if mode == "redis" {
				sem = coalesce.NewSemanticRedis(rdb, redisCfg, th, maxA)
				log.Printf("语义相似合并 semantic=redis（threshold=%v，maxActive=%d）", th, maxA)
			} else {
				sem = coalesce.NewSemanticLocal(mergeTO, th)
				log.Printf("语义相似合并 semantic=local（threshold=%v）", th)
			}
		}
	}

	deps := &httpnb.Deps{
		Exact:         coord,
		Regex:         regexCoord,
		Embedder:      embedSvc,
		Downstream:    rag,
		Coalesce:      merger,
		Semantic:      sem,
		VectorEnabled: vectorEnabled,
		Vector:        vecStore,
		DedupVector:   dedupSearch,
		DedupWriter:   dedupWriter,
	}

	handler := httpnb.NewHandler(deps)
	srv := bootstrap.NewHTTPServer(cfg.Server.HTTPAddr, handler)

	log.Printf("网关 HTTP 监听 %s（配置文件: %s，Redis: %s）", cfg.Server.HTTPAddr, *configPath, cfg.Redis.Addr)
	if err := srv.Run(*shutdownGrace); err != nil {
		log.Printf("服务退出: %v", err)
		os.Exit(1)
	}
	log.Println("已优雅退出")
}

func newBreakerFromCfg(c config.CircuitBreakerConfig) *circuitbreaker.Breaker {
	if !c.Enabled {
		return nil
	}
	sec := c.OpenSeconds
	if sec <= 0 {
		sec = 30
	}
	return circuitbreaker.New(c.FailureThreshold, time.Duration(sec)*time.Second)
}

func subscribeExactReload(ctx context.Context, rdb *redis.Client, coord *rules.ExactCoordinator) {
	sub := rdb.Subscribe(ctx, rules.ChannelExact())
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			if err := coord.Reload(context.Background()); err != nil {
				log.Printf("Pub/Sub 触发重载精确规则失败: %v", err)
			}
		}
	}
}

func subscribeRegexReload(ctx context.Context, rdb *redis.Client, coord *rules.RegexCoordinator) {
	sub := rdb.Subscribe(ctx, rules.ChannelRegex())
	defer sub.Close()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg == nil {
				continue
			}
			if err := coord.Reload(context.Background()); err != nil {
				log.Printf("Pub/Sub 触发重载正则规则失败: %v", err)
			}
		}
	}
}
