package embedding

import "context"

// Service 北向编排可注入的侧车嵌入能力（nil 表示关闭）。
type Service interface {
	Embed(ctx context.Context, in EmbedInput) (*EmbedResult, error)
}

// protocolVersion 与契约 UDS_EMBEDDING.md v1.0 一致。
const protocolVersion = 1

// maxPayloadLen 单帧 JSON 最大字节数（不含 4 字节长度前缀）。
const maxPayloadLen = 4_194_304

// Request 为 UDS 请求 JSON（camelCase）。
type Request struct {
	ProtocolVersion int    `json:"protocolVersion"`
	RequestID       string `json:"requestId"`
	Kind            string `json:"kind,omitempty"`
	TraceID         string `json:"traceId,omitempty"`
	Text            string `json:"text,omitempty"`
	Model           string `json:"model,omitempty"`
}

// Response 为 UDS 响应 JSON。
type Response struct {
	ProtocolVersion int             `json:"protocolVersion"`
	RequestID       string          `json:"requestId"`
	Kind            string          `json:"kind,omitempty"`
	Dimensions      int             `json:"dimensions,omitempty"`
	Embedding       []float64       `json:"embedding,omitempty"`
	Model           string          `json:"model,omitempty"`
	ServerVersion   string          `json:"serverVersion,omitempty"`
	Error           *ResponseError  `json:"error,omitempty"`
}

// ResponseError 侧车返回的错误体。
type ResponseError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// EmbedInput 网关侧一次嵌入调用入参。
type EmbedInput struct {
	Text    string
	TraceID string
	Model   string
}

// EmbedResult 嵌入成功结果。
type EmbedResult struct {
	Dimensions int
	Embedding  []float64
	Model      string
}
