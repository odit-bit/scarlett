package cluster

import (
	"context"

	"google.golang.org/grpc"
)

func UnaryLogInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	// log.Println("grpc interceptor:", info.FullMethod)
	// md, ok := metadata.FromIncomingContext(ctx)
	// if !ok {
	// 	log.Println("no metadata")
	// } else {
	// 	log.Println("metadata:", md)
	// }
	m, err := handler(ctx, req)
	return m, err
}
