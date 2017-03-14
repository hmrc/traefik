package accesslog

import (
	"bufio"
	"net"
	"net/http"
)

// captureResponseWriter is a wrapper of type http.ResponseWriter
// that tracks request status and size
type captureResponseWriter struct {
	rw     http.ResponseWriter
	status int
	size   int64
}

func (crw *captureResponseWriter) Header() http.Header {
	return crw.rw.Header()
}

func (crw *captureResponseWriter) Write(b []byte) (int, error) {
	if crw.status == 0 {
		crw.status = http.StatusOK
	}
	size, err := crw.rw.Write(b)
	crw.size += int64(size)
	return size, err
}

func (crw *captureResponseWriter) WriteHeader(s int) {
	crw.rw.WriteHeader(s)
	crw.status = s
}

func (crw *captureResponseWriter) Flush() {
	f, ok := crw.rw.(http.Flusher)
	if ok {
		f.Flush()
	}
}

func (crw *captureResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return crw.rw.(http.Hijacker).Hijack()
}

func (crw *captureResponseWriter) Status() int {
	return crw.status
}

func (crw *captureResponseWriter) Size() int64 {
	return crw.size
}
