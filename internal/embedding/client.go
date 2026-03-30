package embedding

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ClientConfig 连接与超时（由上层从 config.yaml 映射而来）。
type ClientConfig struct {
	Transport string // "unix" 或 "tcp"
	// SocketPath UDS 路径（transport=unix）。
	SocketPath string
	// TCPAddr 如 127.0.0.1:18080（transport=tcp）。
	TCPAddr string
	// Timeout 单次 embed 等调用的默认超时（契约默认 100ms）。
	Timeout time.Duration
	// PingTimeout ping/pong 就绪探测超时（可长于 embed）。
	PingTimeout time.Duration
}

// Client UDS/TCP 帧客户端；同一连接上顺序请求（契约禁止交错）。
type Client struct {
	cfg ClientConfig

	mu   sync.Mutex
	conn net.Conn
}

// NewClient 构造客户端；cfg 由调用方校验 transport 与地址非空。
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 100 * time.Millisecond
	}
	if cfg.PingTimeout <= 0 {
		cfg.PingTimeout = 5 * time.Second
	}
	return &Client{cfg: cfg}
}

func (c *Client) dial() (net.Conn, error) {
	switch c.cfg.Transport {
	case "tcp":
		if c.cfg.TCPAddr == "" {
			return nil, fmt.Errorf("embedding: tcp 模式需配置 tcp_addr")
		}
		return net.DialTimeout("tcp", c.cfg.TCPAddr, 2*time.Second)
	case "unix", "":
		if c.cfg.SocketPath == "" {
			return nil, fmt.Errorf("embedding: unix 模式需配置 socket_path")
		}
		return net.DialTimeout("unix", c.cfg.SocketPath, 2*time.Second)
	default:
		return nil, fmt.Errorf("embedding: 未知 transport %q", c.cfg.Transport)
	}
}

func (c *Client) ensureConn() error {
	if c.conn != nil {
		return nil
	}
	conn, err := c.dial()
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) resetConn() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

// Close 关闭底层连接。
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}
	return err
}

// Ping 发送 ping，期望 pong（用于就绪检查）。
func (c *Client) Ping(ctx context.Context) error {
	req := &Request{
		ProtocolVersion: protocolVersion,
		RequestID:       uuid.NewString(),
		Kind:            "ping",
	}
	return c.roundTrip(ctx, c.cfg.PingTimeout, req, func(res *Response) error {
		if res.Error != nil {
			return &ServerError{Code: res.Error.Code, Message: res.Error.Message}
		}
		if res.Kind != "pong" {
			return fmt.Errorf("embedding: 期望 kind=pong，实际 %q", res.Kind)
		}
		return nil
	})
}

// Embed 计算文本向量；ctx 取消或超时须尽快返回。
func (c *Client) Embed(ctx context.Context, in EmbedInput) (*EmbedResult, error) {
	text := trimEmbedText(in.Text)
	if text == "" {
		return nil, fmt.Errorf("embedding: text 不能为空")
	}
	req := &Request{
		ProtocolVersion: protocolVersion,
		RequestID:       uuid.NewString(),
		Kind:            "embed",
		TraceID:         in.TraceID,
		Text:            text,
		Model:           in.Model,
	}
	var out *EmbedResult
	err := c.roundTrip(ctx, c.cfg.Timeout, req, func(res *Response) error {
		if res.Error != nil {
			return &ServerError{Code: res.Error.Code, Message: res.Error.Message}
		}
		if res.Dimensions <= 0 || len(res.Embedding) == 0 {
			return fmt.Errorf("embedding: 响应缺少 dimensions/embedding")
		}
		if res.Dimensions != len(res.Embedding) {
			return fmt.Errorf("embedding: dimensions 与 embedding 长度不一致")
		}
		out = &EmbedResult{
			Dimensions: res.Dimensions,
			Embedding:  res.Embedding,
			Model:      res.Model,
		}
		return nil
	})
	return out, err
}

func trimEmbedText(s string) string {
	// 简单 trim；与侧车「trim 后 ≥1」对齐。
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r') {
		s = s[1:]
	}
	for len(s) > 0 {
		last := s[len(s)-1]
		if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
			break
		}
		s = s[:len(s)-1]
	}
	return s
}

func (c *Client) roundTrip(ctx context.Context, opTimeout time.Duration, req *Request, onOK func(*Response) error) error {
	payload, err := marshalRequest(req)
	if err != nil {
		return err
	}
	if len(payload) > maxPayloadLen {
		return ErrFrameTooLarge
	}

	deadline := time.Now().Add(opTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	callCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if callCtx.Err() != nil {
		return callCtx.Err()
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConn(); err != nil {
		return err
	}
	_ = c.conn.SetDeadline(deadline)
	if err := writeFrame(c.conn, payload); err != nil {
		c.resetConn()
		return err
	}
	raw, err := readFrame(c.conn)
	if err != nil {
		c.resetConn()
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return fmt.Errorf("embedding: %w", context.DeadlineExceeded)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if callCtx.Err() != nil {
			return callCtx.Err()
		}
		return err
	}
	res, err := unmarshalResponse(raw)
	if err != nil {
		c.resetConn()
		return err
	}
	if res.RequestID != "" && res.RequestID != req.RequestID {
		c.resetConn()
		return fmt.Errorf("embedding: requestId 回显不一致")
	}
	if err := onOK(res); err != nil {
		return err
	}
	return nil
}
