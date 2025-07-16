package grpc_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	cldf_grpc "github.com/smartcontractkit/chainlink-deployments-framework/internal/grpc"
)

// mockInvoker is a mock gRPC invoker for testing interceptors
type mockInvoker struct {
	method      string
	req         any
	reply       any
	cc          *grpc.ClientConn
	opts        []grpc.CallOption
	err         error
	capturedCtx context.Context //nolint:containedctx // needed for test verification
}

func (m *mockInvoker) invoke(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	opts ...grpc.CallOption,
) error {
	m.capturedCtx = ctx
	m.method = method
	m.req = req
	m.reply = reply
	m.cc = cc
	m.opts = opts

	return m.err
}

func TestGapTokenInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		token             string
		expectedHeader    string
		expectedHeaderKey string
	}{
		{
			name:              "valid_token",
			token:             "test-token-123",
			expectedHeader:    "Bearer test-token-123",
			expectedHeaderKey: "x-authorization-github-jwt",
		},
		{
			name:              "empty_token",
			token:             "",
			expectedHeader:    "Bearer ",
			expectedHeaderKey: "x-authorization-github-jwt",
		},
		{
			name:              "token_with_spaces",
			token:             "test token with spaces",
			expectedHeader:    "Bearer test token with spaces",
			expectedHeaderKey: "x-authorization-github-jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock invoker
			mock := &mockInvoker{}
			interceptor := cldf_grpc.GapTokenInterceptor(tt.token)

			// Create test context and connection
			ctx := context.Background()
			conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			// Execute interceptor
			err = interceptor(
				ctx,
				"/test.Service/TestMethod",
				&struct{}{},
				&struct{}{},
				conn,
				mock.invoke,
			)

			// Verify
			require.NoError(t, err)
			require.NotNil(t, mock.capturedCtx)

			// Check that the metadata was added to the context
			md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
			require.True(t, ok, "metadata should be present in context")

			// Verify the authorization header
			values := md.Get(tt.expectedHeaderKey)
			require.Len(t, values, 1, "should have exactly one authorization header")
			require.Equal(t, tt.expectedHeader, values[0])

			// Verify other parameters were passed through correctly
			require.Equal(t, "/test.Service/TestMethod", mock.method)
			require.NotNil(t, mock.req)
			require.NotNil(t, mock.reply)
			require.Equal(t, conn, mock.cc)
		})
	}
}

func TestGapRepositoryInterceptor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		repository        string
		expectedHeader    string
		expectedHeaderKey string
	}{
		{
			name:              "valid_repository",
			repository:        "smartcontractkit/chainlink",
			expectedHeader:    "smartcontractkit/chainlink",
			expectedHeaderKey: "x-repository",
		},
		{
			name:              "empty_repository",
			repository:        "",
			expectedHeader:    "",
			expectedHeaderKey: "x-repository",
		},
		{
			name:              "repository_with_underscores",
			repository:        "smart_contract_kit/chainlink_framework",
			expectedHeader:    "smart_contract_kit/chainlink_framework",
			expectedHeaderKey: "x-repository",
		},
		{
			name:              "repository_with_dots",
			repository:        "github.com/smartcontractkit/chainlink",
			expectedHeader:    "github.com/smartcontractkit/chainlink",
			expectedHeaderKey: "x-repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock invoker
			mock := &mockInvoker{}
			interceptor := cldf_grpc.GapRepositoryInterceptor(tt.repository)

			// Create test context and connection
			ctx := context.Background()
			conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			// Execute interceptor
			err = interceptor(
				ctx,
				"/test.Service/TestMethod",
				&struct{}{},
				&struct{}{},
				conn,
				mock.invoke,
			)

			// Verify
			require.NoError(t, err)
			require.NotNil(t, mock.capturedCtx)

			// Check that the metadata was added to the context
			md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
			require.True(t, ok, "metadata should be present in context")

			// Verify the repository header
			values := md.Get(tt.expectedHeaderKey)
			require.Len(t, values, 1, "should have exactly one repository header")
			require.Equal(t, tt.expectedHeader, values[0])

			// Verify other parameters were passed through correctly
			require.Equal(t, "/test.Service/TestMethod", mock.method)
			require.NotNil(t, mock.req)
			require.NotNil(t, mock.reply)
			require.Equal(t, conn, mock.cc)
		})
	}
}

func TestGapInterceptors_ErrorPropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		interceptor grpc.UnaryClientInterceptor
		invokerErr  error
	}{
		{
			name:        "token_interceptor_propagates_error",
			interceptor: cldf_grpc.GapTokenInterceptor("test-token"),
			invokerErr:  errors.New("test connection error"),
		},
		{
			name:        "repository_interceptor_propagates_error",
			interceptor: cldf_grpc.GapRepositoryInterceptor("test-repo"),
			invokerErr:  errors.New("test server error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create mock invoker that returns an error
			mock := &mockInvoker{err: tt.invokerErr}

			// Create test context and connection
			ctx := context.Background()
			conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer conn.Close()

			// Execute interceptor
			err = tt.interceptor(
				ctx,
				"/test.Service/TestMethod",
				&struct{}{},
				&struct{}{},
				conn,
				mock.invoke,
			)

			// Verify that the error is propagated
			require.Error(t, err)
			require.Equal(t, tt.invokerErr, err)
		})
	}
}

func TestGapInterceptors_MetadataAccumulation(t *testing.T) {
	t.Parallel()

	// Test that both interceptors can be chained and both add their metadata
	mock := &mockInvoker{}

	tokenInterceptor := cldf_grpc.GapTokenInterceptor("test-token")
	repoInterceptor := cldf_grpc.GapRepositoryInterceptor("test-repo")

	// Create test context and connection
	ctx := context.Background()
	conn, err := grpc.NewClient("localhost:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(tokenInterceptor, repoInterceptor))
	require.NoError(t, err)
	defer conn.Close()

	// Chain the interceptors
	chainedInvoker := func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		opts ...grpc.CallOption,
	) error {
		return repoInterceptor(ctx, method, req, reply, cc, mock.invoke, opts...)
	}

	// Execute the chained interceptors
	err = tokenInterceptor(
		ctx,
		"/test.Service/TestMethod",
		&struct{}{},
		&struct{}{},
		conn,
		chainedInvoker,
	)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, mock.capturedCtx)

	// Check that both metadata headers were added
	md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
	require.True(t, ok, "metadata should be present in context")

	// Verify both headers are present
	tokenValues := md.Get("x-authorization-github-jwt")
	require.Len(t, tokenValues, 1, "should have token header")
	require.Equal(t, "Bearer test-token", tokenValues[0])

	repoValues := md.Get("x-repository")
	require.Len(t, repoValues, 1, "should have repository header")
	require.Equal(t, "test-repo", repoValues[0])
}

func TestGapInterceptors_ExistingMetadata(t *testing.T) {
	t.Parallel()

	// Test that interceptors preserve existing metadata
	mock := &mockInvoker{}
	interceptor := cldf_grpc.GapTokenInterceptor("test-token")

	// Create context with existing metadata
	ctx := context.Background()
	ctx = metadata.AppendToOutgoingContext(ctx, "existing-header", "existing-value")

	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	// Execute interceptor
	err = interceptor(
		ctx,
		"/test.Service/TestMethod",
		&struct{}{},
		&struct{}{},
		conn,
		mock.invoke,
	)

	// Verify
	require.NoError(t, err)
	require.NotNil(t, mock.capturedCtx)

	// Check that both old and new metadata are present
	md, ok := metadata.FromOutgoingContext(mock.capturedCtx)
	require.True(t, ok, "metadata should be present in context")

	// Verify existing metadata is preserved
	existingValues := md.Get("existing-header")
	require.Len(t, existingValues, 1, "should preserve existing header")
	require.Equal(t, "existing-value", existingValues[0])

	// Verify new metadata is added
	tokenValues := md.Get("x-authorization-github-jwt")
	require.Len(t, tokenValues, 1, "should have new token header")
	require.Equal(t, "Bearer test-token", tokenValues[0])
}
