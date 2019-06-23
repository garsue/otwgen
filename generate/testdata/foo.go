package testdata

import (
	"context"
	"io"
	"net/http"
	"os"
)

// noinspection GoUnusedExportedFunction
func MustFoo(ctx context.Context) {
	if err := Foo(ctx); err != nil {
		panic(err)
	}
}

func Foo(ctx context.Context) (err error) {
	request, err := newRequest(ctx)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if err1 := response.Body.Close(); err1 != nil && err == nil {
			err = err1
		}
	}()
	if _, err := io.Copy(os.Stdout, response.Body); err != nil {
		return err
	}
	return nil
}

func newRequest(ctx context.Context) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		return nil, err
	}
	return request.WithContext(ctx), err
}
