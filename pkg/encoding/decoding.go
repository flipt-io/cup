package encoding

import "io"

type AnyDecoder interface {
	Decode(any) error
}

type DecodeBuilder interface {
	NewDecoder(io.Reader) AnyDecoder
}

type Decoder[B DecodeBuilder, T any] struct {
	decoder AnyDecoder
}

func NewDecoder[B DecodeBuilder, T any](r io.Reader) Decoder[B, T] {
	var b DecodeBuilder

	return Decoder[B, T]{decoder: b.NewDecoder(r)}
}

func (d Decoder[B, T]) Decode() (*T, error) {
	var t T
	return &t, d.decoder.Decode(&t)
}
