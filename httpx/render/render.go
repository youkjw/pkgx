package render

import "net/http"

type Render interface {
	Parse() []byte
	WriterContentType(w http.ResponseWriter)
}

func writeContentType(w http.ResponseWriter, value []string) {
	header := w.Header()
	if val := header["Content-Type"]; len(val) == 0 {
		for _, v := range value {
			header.Add("Content-Type", v)
		}
	}
}
