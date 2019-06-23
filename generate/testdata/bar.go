package testdata

import (
	"context"
)

type Bar struct {
}

func (b Bar) MustMethod(ctx context.Context) {
	if err := b.Method(ctx); err != nil {
		panic(err)
	}
}

func (b Bar) Method(ctx context.Context) error {
	return Foo(ctx)
}
