package arpc

import "net/http"

type SSEResponseWriter interface {
	http.ResponseWriter
	http.Flusher
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

func (w *sseResponseWriter) Flush() {
	w.w.(http.Flusher).Flush()
}
