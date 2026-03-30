package embedding

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServer 单连接处理一帧请求并回复一帧响应（TCP 联调替身）。
func fakeServer(t *testing.T, ln net.Listener, handler func(*Request) *Response) {
	t.Helper()
	conn, err := ln.Accept()
	require.NoError(t, err)
	defer conn.Close()
	raw, err := readFrame(conn)
	require.NoError(t, err)
	var req Request
	require.NoError(t, json.Unmarshal(raw, &req))
	res := handler(&req)
	b, err := json.Marshal(res)
	require.NoError(t, err)
	require.NoError(t, writeFrame(conn, b))
}

func TestClient_Ping_TCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go fakeServer(t, ln, func(req *Request) *Response {
		assert.Equal(t, "ping", req.Kind)
		return &Response{ProtocolVersion: 1, RequestID: req.RequestID, Kind: "pong", ServerVersion: "test"}
	})

	addr := ln.Addr().String()
	c := NewClient(ClientConfig{Transport: "tcp", TCPAddr: addr, PingTimeout: time.Second, Timeout: time.Second})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, c.Ping(ctx))
}

func TestClient_Embed_TCP(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go fakeServer(t, ln, func(req *Request) *Response {
		assert.Equal(t, "embed", req.Kind)
		assert.NotEmpty(t, req.Text)
		return &Response{
			ProtocolVersion: 1,
			RequestID:       req.RequestID,
			Dimensions:      2,
			Model:           "m",
			Embedding:       []float64{0.1, 0.2},
		}
	})

	addr := ln.Addr().String()
	c := NewClient(ClientConfig{Transport: "tcp", TCPAddr: addr, Timeout: time.Second, PingTimeout: time.Second})
	defer c.Close()

	res, err := c.Embed(context.Background(), EmbedInput{Text: "  hello  ", TraceID: "trace"})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, 2, res.Dimensions)
	assert.Equal(t, []float64{0.1, 0.2}, res.Embedding)
	assert.Equal(t, "m", res.Model)
}

func TestClient_Embed_ServerError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go fakeServer(t, ln, func(req *Request) *Response {
		return &Response{
			ProtocolVersion: 1,
			RequestID:       req.RequestID,
			Error:           &ResponseError{Code: "VALIDATION_ERROR", Message: "empty"},
		}
	})

	c := NewClient(ClientConfig{Transport: "tcp", TCPAddr: ln.Addr().String(), Timeout: time.Second})
	defer c.Close()

	_, err = c.Embed(context.Background(), EmbedInput{Text: "x"})
	var se *ServerError
	require.ErrorAs(t, err, &se)
	assert.Equal(t, "VALIDATION_ERROR", se.Code)
}

func TestClient_Embed_Timeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, aerr := ln.Accept()
		if aerr != nil {
			return
		}
		defer conn.Close()
		_, _ = readFrame(conn)
		time.Sleep(500 * time.Millisecond)
	}()

	c := NewClient(ClientConfig{Transport: "tcp", TCPAddr: ln.Addr().String(), Timeout: 50 * time.Millisecond})
	defer c.Close()

	ctx := context.Background()
	_, err = c.Embed(ctx, EmbedInput{Text: "slow"})
	assert.Error(t, err)
}
