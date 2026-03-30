package circuitbreaker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBreaker_OpenAfterFailures(t *testing.T) {
	b := New(3, 50*time.Millisecond)
	assert.True(t, b.Allow())
	b.Fail()
	b.Fail()
	assert.True(t, b.Allow())
	b.Fail()
	assert.False(t, b.Allow())
	assert.False(t, b.Allow())
}

func TestBreaker_RecoverAfterWindow(t *testing.T) {
	b := New(2, 30*time.Millisecond)
	assert.True(t, b.Allow())
	b.Fail()
	b.Fail()
	assert.False(t, b.Allow())

	time.Sleep(40 * time.Millisecond)
	assert.True(t, b.Allow())
	b.Success()
	assert.True(t, b.Allow())
}

func TestBreaker_SuccessResets(t *testing.T) {
	b := New(5, time.Second)
	b.Fail()
	b.Fail()
	b.Success()
	b.Fail()
	b.Fail()
	b.Fail()
	b.Fail()
	assert.True(t, b.Allow())
}
