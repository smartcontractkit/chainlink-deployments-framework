package jd

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func authTokenInterceptor(source oauth2.TokenSource) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		token, err := source.Token()
		if err != nil {
			return err
		}

		return invoker(
			metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token.AccessToken),
			method, req, reply, cc, opts...,
		)
	}
}
