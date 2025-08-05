package jd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Mock TokenSource for testing OAuth2 functionality
type MockTokenSource struct {
	mock.Mock
}

func (m *MockTokenSource) Token() (*oauth2.Token, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*oauth2.Token), args.Error(1)
}

// Mock UnaryInvoker for testing interceptor behavior
type MockUnaryInvoker struct {
	mock.Mock
	capturedCtx context.Context //nolint:containedctx // Needed to capture modified context in tests
}

func (m *MockUnaryInvoker) Invoke(
	ctx context.Context,
	method string,
	req, reply any,
	cc *grpc.ClientConn,
	opts ...grpc.CallOption,
) error {
	m.capturedCtx = ctx // Capture the context to verify metadata
	args := m.Called(ctx, method, req, reply, cc, opts)

	return args.Error(0)
}

func TestAuthTokenInterceptor_Success(t *testing.T) {
	t.Parallel()

	// Setup
	mockTokenSource := &MockTokenSource{}
	expectedToken := &oauth2.Token{
		AccessToken: "test-access-token-123",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(time.Hour),
	}
	mockTokenSource.On("Token").Return(expectedToken, nil)

	mockInvoker := &MockUnaryInvoker{}
	mockInvoker.On("Invoke", mock.Anything, "test.method", "request", "reply", (*grpc.ClientConn)(nil), []grpc.CallOption(nil)).Return(nil)

	// Create interceptor
	interceptor := authTokenInterceptor(mockTokenSource)

	// Execute
	ctx := context.Background()
	err := interceptor(ctx, "test.method", "request", "reply", nil, mockInvoker.Invoke)

	// Verify
	require.NoError(t, err)
	mockTokenSource.AssertExpectations(t)
	mockInvoker.AssertExpectations(t)

	// Verify that the authorization header was added to the context
	md, exists := metadata.FromOutgoingContext(mockInvoker.capturedCtx)
	require.True(t, exists)
	authHeaders := md.Get("authorization")
	require.Len(t, authHeaders, 1)
	assert.Equal(t, "Bearer test-access-token-123", authHeaders[0])
}

func TestAuthTokenInterceptor_TokenError(t *testing.T) {
	t.Parallel()

	// Setup
	mockTokenSource := &MockTokenSource{}
	expectedError := errors.New("token retrieval failed")
	mockTokenSource.On("Token").Return(nil, expectedError)

	mockInvoker := &MockUnaryInvoker{} // Should not be called

	// Create interceptor
	interceptor := authTokenInterceptor(mockTokenSource)

	// Execute
	ctx := context.Background()
	err := interceptor(ctx, "test.method", "request", "reply", nil, mockInvoker.Invoke)

	// Verify
	require.Error(t, err)
	assert.Equal(t, expectedError, err)
	mockTokenSource.AssertExpectations(t)
	mockInvoker.AssertNotCalled(t, "Invoke")
}

func TestInterceptors_CallOptionsPreservation(t *testing.T) {
	t.Parallel()

	// Test that call options are properly passed through interceptors
	t.Run("authTokenInterceptor", func(t *testing.T) {
		t.Parallel()

		// Setup
		mockTokenSource := &MockTokenSource{}
		mockTokenSource.On("Token").Return(&oauth2.Token{AccessToken: "test"}, nil)

		mockInvoker := &MockUnaryInvoker{}
		expectedOpts := []grpc.CallOption{grpc.WaitForReady(true)}
		mockInvoker.On("Invoke", mock.Anything, "test.method", "request", "reply", (*grpc.ClientConn)(nil), expectedOpts).Return(nil)

		interceptor := authTokenInterceptor(mockTokenSource)

		// Execute
		ctx := context.Background()
		err := interceptor(ctx, "test.method", "request", "reply", nil, mockInvoker.Invoke, expectedOpts...)

		// Verify
		require.NoError(t, err)
		mockInvoker.AssertExpectations(t)
		mockTokenSource.AssertExpectations(t)
	})
}
