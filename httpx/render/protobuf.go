package render

import (
	"google.golang.org/protobuf/proto"
	"net/http"
)

const (
	ProtobufHeaderContentType = iota
	StreamHeaderContentType
)

var protobufContentType = []string{"application/x-protobuf"}
var streamContentType = []string{"application/octet-stream"}

type Protobuf struct {
	contentType int8
	Data        proto.Message
}

func RenderProtobuf(data proto.Message, contentType int8) *Protobuf {
	return &Protobuf{
		contentType: contentType,
		Data:        data,
	}
}

func (r *Protobuf) Parse() []byte {
	protoBytes, err := proto.Marshal(r.Data)
	if err != nil {
		panic(err)
	}
	return protoBytes
}

func (r *Protobuf) WriterContentType(w http.ResponseWriter) {
	contentType := protobufContentType
	if r.contentType == ProtobufHeaderContentType {
		contentType = streamContentType
	}

	writeContentType(w, contentType)
}
