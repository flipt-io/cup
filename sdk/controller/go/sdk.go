package sdk

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.flipt.io/cup/pkg/encoding"
)

var ErrNotFound = errors.New("not found")

type CLI struct {
	kinds map[string]Runner
}

func NewCLI() *CLI {
	return &CLI{kinds: map[string]Runner{}}
}

func (c *CLI) Run(ctx context.Context, args ...string) {
	if len(args) < 3 {
		fmt.Fprintf(os.Stderr, "too few arguments: count %d\n", len(args))
		os.Exit(1)
	}

	kind, ok := c.kinds[os.Args[2]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unsupported kind: %q\n", os.Args[2])
		os.Exit(1)
	}

	if err := kind.Run(ctx, args...); err != nil {
		code := 1
		if errors.Is(err, ErrNotFound) {
			code = 2
		}

		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(code)
	}
}

func (c *CLI) RegisterKind(kind string, run Runner) {
	if c.kinds == nil {
		c.kinds = map[string]Runner{}
	}

	c.kinds[kind] = run
}

type Runner interface {
	Run(ctx context.Context, args ...string) error
}

type Kind[T any] struct {
	runtime KindController[T]
}

func NewKindController[T any](runtime KindController[T]) Kind[T] {
	return Kind[T]{
		runtime: runtime,
	}
}

func (c Kind[T]) Run(ctx context.Context, args ...string) error {
	enc := encoding.NewJSONEncoder[T](os.Stdout)
	switch args[1] {
	case "get":
		return c.runtime.Get(ctx, args[3], args[4], enc)
	case "list":
		return c.runtime.List(ctx, args[3], enc)
	case "put":
		t, err := encoding.NewJSONDecoder[T](os.Stdin).Decode()
		if err != nil {
			return err
		}

		return c.runtime.Put(ctx, args[3], args[4], t)
	case "delete":
		if len(args) < 5 {
			panic("delete <kind> <namespace> <name>")
		}

		return c.runtime.Delete(ctx, args[3], args[4])
	default:
		return fmt.Errorf("unexpected command: %q", args[1])
	}
}

type KindController[T any] interface {
	Get(ctx context.Context, namespace, name string, enc encoding.TypedEncoder[T]) error
	List(ctx context.Context, namespace string, enc encoding.TypedEncoder[T]) error
	Put(ctx context.Context, namespace, name string, t *T) error
	Delete(ctx context.Context, namespace, name string) error
}
