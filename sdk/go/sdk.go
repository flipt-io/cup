package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go.flipt.io/cup"
)

type CLI struct {
	typ     cup.Type
	runtime Runtime
}

func New(typ cup.Type, runtime Runtime) CLI {
	return CLI{
		typ:     typ,
		runtime: runtime,
	}
}

func (c CLI) Run(ctx context.Context, args ...string) error {
	enc := json.NewEncoder(os.Stdout)
	switch args[1] {
	case "type":
		return enc.Encode(c.typ)
	case "list":
		return c.runtime.ListAll(ctx, enc)
	case "put":
		return c.runtime.Put(ctx, os.Stdin, enc)
	case "delete":
		if len(args) < 4 {
			panic("delete [namespace] [id]")
		}

		return c.runtime.Delete(ctx, cup.Namespace(args[2]), cup.ID(args[3]), enc)
	default:
		return fmt.Errorf("unexpected command: %q", args[1])
	}
}

type TypedRuntime[T any] interface {
	ListAll(ctx context.Context, enc TypedEncoder[T]) error
	Put(ctx context.Context, t *T, enc TypedEncoder[cup.Change]) error
	Delete(ctx context.Context, namespace cup.Namespace, id cup.ID, enc TypedEncoder[cup.Change]) error
}

type Encoder interface {
	Encode(any) error
}

type Runtime interface {
	ListAll(ctx context.Context, enc Encoder) error
	Put(ctx context.Context, r io.Reader, enc Encoder) error
	Delete(ctx context.Context, namespace cup.Namespace, id cup.ID, enc Encoder) error
}

type TypedEncoder[T any] struct {
	enc Encoder
}

func (e TypedEncoder[T]) Encode(t T) error {
	return e.enc.Encode(t)
}

type runtime struct {
	listAll func(context.Context, Encoder) error
	put     func(context.Context, io.Reader, Encoder) error
	del     func(context.Context, cup.Namespace, cup.ID, Encoder) error
}

func Typed[T any](run TypedRuntime[T]) Runtime {
	return &runtime{
		listAll: func(ctx context.Context, enc Encoder) error {
			return run.ListAll(ctx, TypedEncoder[T]{enc: enc})
		},
		put: func(ctx context.Context, r io.Reader, enc Encoder) error {
			var t T
			if err := json.NewDecoder(r).Decode(&t); err != nil {
				return nil
			}

			return run.Put(ctx, &t, TypedEncoder[cup.Change]{enc: enc})
		},
		del: func(ctx context.Context, ns cup.Namespace, id cup.ID, enc Encoder) error {
			return run.Delete(ctx, ns, id, TypedEncoder[cup.Change]{enc: enc})
		},
	}
}

func (r *runtime) ListAll(ctx context.Context, enc Encoder) error {
	return r.listAll(ctx, enc)
}

func (r *runtime) Put(ctx context.Context, rd io.Reader, enc Encoder) error {
	return r.put(ctx, rd, enc)
}

func (r *runtime) Delete(ctx context.Context, namespace cup.Namespace, id cup.ID, enc Encoder) error {
	return r.del(ctx, namespace, id, enc)
}
