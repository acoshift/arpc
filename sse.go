package arpc

import (
	"net/http"
	"strings"
)

type SSEResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	WriteEvent(event, data string) error
	WriteData(data string) error
}

var _ SSEResponseWriter = (*sseResponseWriter)(nil)

type sseResponseWriter struct {
	wrote bool
	w     http.ResponseWriter
}

func newSSEResponseWriter(w http.ResponseWriter) SSEResponseWriter {
	return &sseResponseWriter{w: w}
}

func (w *sseResponseWriter) writeHeader() {
	if w.wrote {
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (w *sseResponseWriter) Header() http.Header {
	return w.w.Header()
}

func (w *sseResponseWriter) WriteHeader(statusCode int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.w.Header().Set("Content-Type", "text/event-stream")
	w.w.WriteHeader(statusCode)
}

func (w *sseResponseWriter) Write(b []byte) (int, error) {
	w.writeHeader()
	return w.w.Write(b)
}

func (w *sseResponseWriter) writeData(data string) error {
	for _, x := range strings.Split(data, "\n") {
		_, err := w.w.Write([]byte("data: " + x + "\n"))
		if err != nil {
			return err
		}
	}
	_, err := w.w.Write([]byte("\n"))
	return err
}

func (w *sseResponseWriter) WriteEvent(event, data string) error {
	w.writeHeader()
	_, err := w.w.Write([]byte("event: " + event + "\n"))
	if err != nil {
		return err
	}
	return w.writeData(data)
}

func (w *sseResponseWriter) WriteData(data string) error {
	w.writeHeader()
	return w.writeData(data)
}

func (w *sseResponseWriter) Flush() {
	w.w.(http.Flusher).Flush()
}
