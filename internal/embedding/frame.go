package embedding

import (
	"encoding/binary"
	"encoding/json"
	"io"
)

// writeFrame 写入一帧：4 字节大端长度 + UTF-8 JSON。
func writeFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxPayloadLen {
		return ErrFrameTooLarge
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

// readFrame 读取一帧；返回 payload 副本，调用方可归还 buffer 池（若从池取用）。
func readFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 {
		return nil, ErrEmptyResponse
	}
	if int64(n) > maxPayloadLen {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func marshalRequest(req *Request) ([]byte, error) {
	return json.Marshal(req)
}

func unmarshalResponse(payload []byte) (*Response, error) {
	var res Response
	if err := json.Unmarshal(payload, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
