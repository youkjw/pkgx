package render

import "net/http"

var plainContentType = []string{"text/plain; charset=utf8"}

type Plain struct {
	Data []byte
}

func RenderPlain(data []byte) *Plain {
	return &Plain{
		Data: data,
	}
}

func (r *Plain) Parse() []byte {
	return r.Data
}

func (r *Plain) WriterContentType(w http.ResponseWriter) {
	writeContentType(w, plainContentType)
}
