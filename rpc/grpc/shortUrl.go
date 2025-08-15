package grpc

import (
	"context"
	"google.golang.org/grpc"
	short_url_v2 "short_url_rpc_study/proto/short_url/v1"
	"short_url_rpc_study/rpc/service"
)

type ShortUrlServiceServer struct {
	short_url_v2.UnimplementedShortUrlServiceServer
	svc service.ShortUrlService
}

func NewShortUrlServiceServer(svc service.ShortUrlService) *ShortUrlServiceServer {
	return &ShortUrlServiceServer{svc: svc}
}

func (s *ShortUrlServiceServer) Register(server grpc.ServiceRegistrar) {
	short_url_v2.RegisterShortUrlServiceServer(server, s)
}

func (s *ShortUrlServiceServer) GenerateShortUrl(ctx context.Context, req *short_url_v2.GenerateShortUrlRequest) (*short_url_v2.GenerateShortUrlResponse, error) {
	shortUrl, err := s.svc.Create(ctx, req.GetOriginUrl())
	if err != nil {
		return nil, err
	}
	return &short_url_v2.GenerateShortUrlResponse{ShortUrl: shortUrl}, nil
}

func (s *ShortUrlServiceServer) GetOriginUrl(ctx context.Context, req *short_url_v2.GetOriginUrlRequest) (*short_url_v2.GetOriginUrlResponse, error) {
	originUrl, err := s.svc.Redirect(ctx, req.GetShortUrl())
	if err != nil {
		return nil, err
	}
	return &short_url_v2.GetOriginUrlResponse{OriginUrl: originUrl}, nil
}
