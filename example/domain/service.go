//go:generate otwgen generate -d ../wrapper github.com/garsue/otwgen/example/domain
package domain

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
	defer func() {
		if err1 := resp.Body.Close(); err1 != nil && err == nil {
			err = err1
		}
	}()
	return resp.Header, nil
}
