# otwgen

OpenCensus embedded wrapper generator.

## Feature

- Detect traceable functions/methods.
- Generate wrapper functions/methods. Wrapper set a span for tracing and just call original functions/methods.

## Usage

Write `go generate` to the file which you want to generate wrapper.

```go
//go:generate otwgen generate -d ./wrapper github.com/xxx/foo
package foo

import (
	"context"
	"net/http"
)

func NewService() *Service {
	return &Service{}
}

type Service struct {
	client http.Client
}

func (s *Service) GetContent(ctx context.Context) (header http.Header, err error) {
	req, err := http.NewRequest(http.MethodHead, "http://example.com", nil)
    if err != nil {
        return nil, err
    }
    req = req.WithContext(ctx)
    resp, err := s.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    return resp.Header, nil
}
```

Then, the wrapper is generated.


```go
package foo

import (
	"context"
	"github.com/xxx/foo"
	"go.opencensus.io/trace"
	"net/http"
)

type Service struct {
	*foo.Service
}

func NewService(orig *foo.Service) *Service {
	return &Service{orig}
}
func (r *Service) GetContent(ctx context.Context) (header http.Header, err error) {
	ctx, span := trace.StartSpan(ctx, "auto generated span")
	defer span.End()
	return r.Service.GetContent(ctx)
}
```