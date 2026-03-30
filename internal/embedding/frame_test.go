package embedding

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadFrame_RoundTrip(t *testing.T) {
	req := &Request{ProtocolVersion: 1, RequestID: "rid", Kind: "ping"}
	payload, err := marshalRequest(req)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, writeFrame(&buf, payload))
	out, err := readFrame(&buf)
	require.NoError(t, err)
	assert.Equal(t, payload, out)
}

func TestReadFrame_TooLarge(t *testing.T) {
	hdr := []byte{0x00, 0x40, 0x00, 0x01} // 4MiB+1
	var buf bytes.Buffer
	_, _ = buf.Write(hdr)
	_, err := readFrame(&buf)
	assert.ErrorIs(t, err, ErrFrameTooLarge)
}

func TestWriteFrame_PayloadTooLarge(t *testing.T) {
	payload := make([]byte, maxPayloadLen+1)
	var buf bytes.Buffer
	err := writeFrame(&buf, payload)
	assert.ErrorIs(t, err, ErrFrameTooLarge)
}

func TestUnmarshalResponse_Error(t *testing.T) {
	raw := []byte(`{"protocolVersion":1,"requestId":"x","error":{"code":"VALIDATION_ERROR","message":"bad"}}`)
	res, err := unmarshalResponse(raw)
	require.NoError(t, err)
	require.NotNil(t, res.Error)
	assert.Equal(t, "VALIDATION_ERROR", res.Error.Code)
}

func TestMarshalRequest_EmbedOmitsEmptyKind(t *testing.T) {
	req := &Request{ProtocolVersion: 1, RequestID: "u", Text: "a"}
	b, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"kind":""`)
}
