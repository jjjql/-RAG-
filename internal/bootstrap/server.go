package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

// HTTPServer 封装可优雅停机的 HTTP 服务。
type HTTPServer struct {
	srv *http.Server
}

// NewHTTPServer 使用 addr 与 handler 构造服务（handler 通常为 northbound 路由）。
func NewHTTPServer(addr string, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run 监听并在收到 SIGINT/SIGTERM 时优雅关闭，超时 shutdownTimeout。
func (s *HTTPServer) Run(shutdownTimeout time.Duration) error {
	if s == nil || s.srv == nil {
		return errors.New("bootstrap: server 未初始化")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := s.srv.Shutdown(shCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("listen: %w", err)
		}
		return nil
	}
}
