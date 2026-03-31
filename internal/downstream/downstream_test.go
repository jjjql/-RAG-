package downstream

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Complete_Mock(t *testing.T) {
	c := &Client{
		A:       NewMock("hello-rag"),
		Timeout: time.Second,
	}
	out, err := c.Complete(context.Background(), AnswerInput{Query: "q"})
	require.NoError(t, err)
	assert.Equal(t, "hello-rag", out)
}

func TestClient_Complete_Disabled(t *testing.T) {
	var c *Client
	_, err := c.Complete(context.Background(), AnswerInput{})
	assert.ErrorIs(t, err, ErrDisabled)
}

func TestClient_Complete_Timeout(t *testing.T) {
	m := &Mock{Text: "x", Delay: 500 * time.Millisecond}
	c := &Client{A: m, Timeout: 20 * time.Millisecond}
	_, err := c.Complete(context.Background(), AnswerInput{Query: "q"})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestMock_DefaultText(t *testing.T) {
	m := NewMock("")
	ctx := context.Background()
	s, err := m.Answer(ctx, AnswerInput{})
	require.NoError(t, err)
	assert.Equal(t, "mock-rag", s)
}
