package render

import (
	"encoding/json"
	eutils "gitlab.cpp32.com/backend/epkg/utils"
	"net/http"
	"strconv"
)

var (
	jsonContextType      = []string{"application/json; charset=utf8"}
	jsonASCIIContentType = []string{"application/json"}
)

type Json struct {
	Data any
}

func RenderJson(data any) *Json {
	return &Json{
		Data: data,
	}
}

func (r *Json) Parse() []byte {
	jsonBytes, err := parseJSON(r.Data)
	if err != nil {
		panic(err)
	}
	return jsonBytes
}

func (r *Json) WriterContentType(w http.ResponseWriter) {
	writeContentType(w, jsonContextType)
}

type UnicodeJson struct {
	Data any
}

func RenderUnicodeJson(data any) *UnicodeJson {
	return &UnicodeJson{
		Data: data,
	}
}

func (r *UnicodeJson) Parse() []byte {
	jsonByte, err := parseJSON(r.Data)
	if err != nil {
		panic(err)
	}
	// 转unicode
	jsonQuoted := strconv.QuoteToASCII(eutils.BytesToString(jsonByte))
	jsonUnquoted := jsonQuoted[1 : len(jsonQuoted)-1]
	return eutils.StringToBytes(jsonUnquoted)
}

func (r *UnicodeJson) WriterContentType(w http.ResponseWriter) {
	writeContentType(w, jsonASCIIContentType)
}

func parseJSON(obj any) ([]byte, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return jsonBytes, err
}
