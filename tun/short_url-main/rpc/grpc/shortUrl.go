package grpc

import (
	"context"
	short_url_v1 "short_url/proto/short_url/v1"
	"short_url/rpc/service"

	"google.golang.org/grpc"
)

type ShortUrlServiceServer struct {
	short_url_v1.UnimplementedShortUrlServiceServer
	svc service.ShortUrlService
}

func NewShortUrlServiceServer(svc service.ShortUrlService) *ShortUrlServiceServer {
	return &ShortUrlServiceServer{svc: svc}
}

func (s *ShortUrlServiceServer) Register(server grpc.ServiceRegistrar) {
	short_url_v1.RegisterShortUrlServiceServer(server, s)
}

func (s *ShortUrlServiceServer) GenerateShortUrl(ctx context.Context, req *short_url_v1.GenerateShortUrlRequest) (*short_url_v1.GenerateShortUrlResponse, error) {
	shortUrl, err := s.svc.Create(ctx, req.GetOriginUrl())
	if err != nil {
		return nil, err
	}
	return &short_url_v1.GenerateShortUrlResponse{ShortUrl: shortUrl}, nil
}

func (s *ShortUrlServiceServer) GetOriginUrl(ctx context.Context, req *short_url_v1.GetOriginUrlRequest) (*short_url_v1.GetOriginUrlResponse, error) {
	originUrl, err := s.svc.Redirect(ctx, req.GetShortUrl())
	if err != nil {
		return nil, err
	}
	return &short_url_v1.GetOriginUrlResponse{OriginUrl: originUrl}, nil
}
