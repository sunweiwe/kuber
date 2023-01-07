package stream

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type WriterFlusher interface {
	http.Flusher
	io.Writer
}

type Pusher struct {
	encoder *json.Encoder
	dst     WriterFlusher
}

func StartPusher(w http.ResponseWriter) (*Pusher, error) {
	flusher, ok := w.(WriterFlusher)
	if !ok {
		return nil, errors.New("not a flushable writer")
	}

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return &Pusher{dst: flusher, encoder: json.NewEncoder(flusher)}, nil
}

func (p *Pusher) Push(data interface{}) error {
	if err := p.encoder.Encode(data); err != nil {
		return err
	}
	p.dst.Flush()
	return nil
}
