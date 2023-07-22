package encoding

import (
	"io"
)

type AnyEncoder interface {
	Encode(any) error
}

type EncoderBuilder interface {
	NewEncoder(io.Writer) AnyEncoder
}

type Encoder[B EncoderBuilder, T any] struct {
	encoder AnyEncoder
}

func NewEncoder[B EncoderBuilder, T any](w io.Writer) Encoder[B, T] {
	var b EncoderBuilder

	return Encoder[B, T]{encoder: b.NewEncoder(w)}
}

func (e Encoder[B, T]) Encode(t *T) error {
	return e.encoder.Encode(t)
}

type AnyEncodingBuilder interface {
	Extension() string
	EncoderBuilder
	DecodeBuilder
}

type TypedEncoder[T any] interface {
	Encode(*T) error
}

type TypedDecoder[T any] interface {
	Decode() (*T, error)
}

type EncodingBuilder[B AnyEncodingBuilder, T any] struct {
	b B
	Encoder[B, T]
	Decoder[B, T]
}

func (e EncodingBuilder[B, T]) Extension() string {
	return e.b.Extension()
}

func (e EncodingBuilder[B, T]) NewEncoder(w io.Writer) TypedEncoder[T] {
	return NewEncoder[B, T](w)
}

func (e EncodingBuilder[B, T]) NewDecoder(r io.Reader) TypedDecoder[T] {
	return NewDecoder[B, T](r)
}
