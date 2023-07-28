package encoding

import (
	"encoding/json"
	"io"
)

type JSON[T any] struct {
	*json.Encoder
	*json.Decoder
}

type JSONEncoding[T any] struct{}

func (j JSONEncoding[T]) Extension() string {
	return "json"
}

func (j *JSONEncoding[T]) NewEncoder(r io.Writer) TypedEncoder[T] {
	return NewJSONEncoder[T](r)
}

func (j *JSONEncoding[T]) NewDecoder(w io.Reader) TypedDecoder[T] {
	return NewJSONDecoder[T](w)
}

func NewJSONEncoder[T any](w io.Writer) *JSON[T] {
	enc := json.NewEncoder(w)
	return &JSON[T]{Encoder: enc}
}

func NewJSONDecoder[T any](r io.Reader) *JSON[T] {
	dec := json.NewDecoder(r)
	return &JSON[T]{Decoder: dec}
}

func (j JSON[T]) Encode(t *T) error {
	return j.Encoder.Encode(t)
}

func (j JSON[T]) Decode() (*T, error) {
	var t T
	return &t, j.Decoder.Decode(&t)
}
