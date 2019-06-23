package testdata

import (
	"context"
	"log"
)

type Bar struct {
	bar
}

type bar struct {
}

func (b *bar) Method(ctx context.Context) error {
	return Foo(ctx)
}

func (b *Bar) MustMethod(ctx context.Context) {
	if err := b.Method(ctx); err != nil {
		b.logError(err)
	}
}

func (b *Bar) Method(ctx context.Context) error {
	return b.bar.Method(ctx)
}

func (b *Bar) logError(err error) {
	log.Println(err)
}
