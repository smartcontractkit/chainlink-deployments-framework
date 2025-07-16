package grpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func GapTokenInterceptor(token string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(
			metadata.AppendToOutgoingContext(ctx, "x-authorization-github-jwt", "Bearer "+token),
			method, req, reply, cc, opts...,
		)
	}
}

func GapRepositoryInterceptor(repository string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		return invoker(
			metadata.AppendToOutgoingContext(ctx, "x-repository", repository),
			method, req, reply, cc, opts...,
		)
	}
}
