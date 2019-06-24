package testdata

import (
	"context"
	aliasedIO "io"
	"net/http"
	"os"
)

// noinspection GoUnusedExportedFunction
func MustFoo(ctx context.Context) {
	if err := Foo(ctx, nil); err != nil {
		panic(err)
	}
}

func Foo(ctx context.Context, body aliasedIO.Reader) (err error) {
	request, err := newRequest(ctx, body)
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
	return Show(response.Body)
}

func Send(ctx context.Context, request *http.Request) (err error) {
	request = request.WithContext(ctx)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if err1 := response.Body.Close(); err1 != nil && err == nil {
			err = err1
		}
	}()
	return Show(response.Body)
}

func Show(body aliasedIO.Reader) error {
	_, err := aliasedIO.Copy(os.Stdout, body)
	return err
}

func newRequest(ctx context.Context, body aliasedIO.Reader) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, "http://localhost", body)
	if err != nil {
		return nil, err
	}
	return request.WithContext(ctx), err
}
