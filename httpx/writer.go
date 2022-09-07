package httpx

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync"
)

const (
	noWritten     = -1
	defaultStatus = http.StatusOK
)

const (
	HeaderContentType               = "Content-type"
	HeaderSetCookie                 = "Set-Cookie"
	HeaderAccessControlAllowOrigin  = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders = "Access-Control-Allow-Headers"

	ContentTypeTextPlain           = "text/plain"
	ContentTypeTextHtml            = "text/html"
	ContentTypeTextXml             = "text/xml"
	ContentTypeApplicationForm     = "application/x-www-form-urlencoded"
	ContentTypeApplicationXml      = "application/xml"
	ContentTypeApplicationJson     = "application/json"
	ContentTypeApplicationProtobuf = "application/x-protobuf"
	ContentTypeApplicationPdf      = "application/pdf"
	ContentTypeApplicationStream   = "application/octet-stream"
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.Flusher

	reset(http.ResponseWriter)
	Status() int
	Size() int
	WriteString(string) (int, error)
	Written() bool
	WriteHeaderNow()
}

var _ ResponseWriter = &ehttpWriter{}

type ehttpWriter struct {
	http.ResponseWriter
	size   int
	status int
	mu     sync.RWMutex
}

func (w *ehttpWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.size = noWritten
	w.status = defaultStatus
}

func (w *ehttpWriter) Status() int {
	return w.status
}

func (w *ehttpWriter) Size() int {
	return w.size
}

func (w *ehttpWriter) Written() bool {
	return w.Size() != noWritten
}

func (w *ehttpWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		w.status = code
	}
}

func (w *ehttpWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *ehttpWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return
}

func (w *ehttpWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		w.ResponseWriter.WriteHeader(w.status)
	}
}

// Hijack implements the http.Hijacker interface.
func (w *ehttpWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.Size() < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// Flush implements the http.Flusher interface.
func (w *ehttpWriter) Flush() {
	w.WriteHeaderNow()
	w.ResponseWriter.(http.Flusher).Flush()
}
