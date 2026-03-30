package embedding

import (
	"errors"
	"fmt"
)

var (
	// ErrFrameTooLarge 帧超过契约上限。
	ErrFrameTooLarge = errors.New("embedding: 帧超过 4MiB 上限")
	// ErrEmptyResponse 对端关闭或读到空帧。
	ErrEmptyResponse = errors.New("embedding: 空响应")
)

// ServerError 侧车 JSON error 字段映射。
type ServerError struct {
	Code    string
	Message string
}

func (e *ServerError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("embedding: 侧车错误 %s: %s", e.Code, e.Message)
}
