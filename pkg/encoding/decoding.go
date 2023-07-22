package encoding

import (
	"errors"
	"io"
)

type AnyDecoder interface {
	Decode(any) error
}

type DecodeBuilder interface {
	NewDecoder(io.Reader) AnyDecoder
}

type Decoder[B DecodeBuilder, T any] struct {
	decoder AnyDecoder
}

type TypedDecoder[T any] interface {
	Decode() (*T, error)
}

func DecodeAll[T any](dec TypedDecoder[T]) (ts []*T, _ error) {
	for {
		t, err := dec.Decode()
		if err == nil {
			ts = append(ts, t)
			continue
		}

		if !errors.Is(err, io.EOF) {
			return nil, err
		}

		return
	}
}

func NewDecoder[B DecodeBuilder, T any](r io.Reader) Decoder[B, T] {
	var b B

	return Decoder[B, T]{decoder: b.NewDecoder(r)}
}

func (d Decoder[B, T]) Decode() (*T, error) {
	var t T
	return &t, d.decoder.Decode(&t)
}
