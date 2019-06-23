package testdata

import (
	"context"
	"io"
	"net/http"
	"os"
)

func Foo(ctx context.Context) {
	request, err := http.NewRequest(http.MethodGet, "http://localhost", nil)
	if err != nil {
		panic(err)
	}
	request = request.WithContext(ctx)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(os.Stdout, response.Body); err != nil {
		panic(err)
	}
	if err := response.Body.Close(); err != nil {
		return
	}
}
