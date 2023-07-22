package encoding

import (
	"encoding/json"
	"io"
)

var _ AnyEncodingBuilder = (*JSON)(nil)

type JSON struct {
	*json.Encoder
	*json.Decoder
}

func NewJSONEncoding[T any]() EncodingBuilder[JSON, T] {
	return EncodingBuilder[JSON, T]{}
}

func (j JSON) Extension() string { return "json" }

func (j JSON) NewEncoder(w io.Writer) AnyEncoder {
	return &JSON{Encoder: json.NewEncoder(w)}
}

func (j JSON) NewDecoder(r io.Reader) AnyDecoder {
	return &JSON{Decoder: json.NewDecoder(r)}
}
