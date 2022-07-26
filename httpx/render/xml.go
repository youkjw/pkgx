package render

import (
	"encoding/xml"
	"net/http"
)

var xmlContentType = []string{"application/xml; charset=utf-8"}

type Xml struct {
	Data any
}

func RenderXml(data any) *Xml {
	return &Xml{
		Data: data,
	}
}

func (r *Xml) Parse() []byte {
	xmlBytes, err := xml.Marshal(r.Data)
	if err != nil {
		panic(err)
	}
	return xmlBytes
}

func (r *Xml) WriterContentType(w http.ResponseWriter) {
	writeContentType(w, plainContentType)
}
