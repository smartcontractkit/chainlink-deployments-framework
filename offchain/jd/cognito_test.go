package jd

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/smartcontractkit/chainlink-deployments-framework/internal/testing/jd/mocks"
)

// newCognitoTokenSourceWithClient creates a new CognitoTokenSource with a custom client.
// This constructor is used for testing with mock clients.
func newCognitoTokenSourceWithClient(auth CognitoAuth, client CognitoClient) *CognitoTokenSource {
	return &CognitoTokenSource{
		auth:   auth,
		client: client,
	}
}

func Test_NewCognitoTokenSource(t *testing.T) {
	t.Parallel()

	auth := CognitoAuth{
		AWSRegion:       "us-east-1",
		AppClientID:     "test-client-id",
		AppClientSecret: "test-client-secret",
		Username:        "testuser",
		Password:        "testpass",
	}

	tokenSource := NewCognitoTokenSource(auth)

	assert.NotNil(t, tokenSource)
	assert.Equal(t, auth, tokenSource.auth)
	assert.Nil(t, tokenSource.client)
	assert.Nil(t, tokenSource.authResult)
}

func Test_CognitoTokenSource_Authenticate(t *testing.T) {
	t.Parallel()

	auth := CognitoAuth{
		AWSRegion:       "us-east-1",
		AppClientID:     "test-client-id",
		AppClientSecret: "test-client-secret",
		Username:        "testuser",
		Password:        "testpass",
	}

	tests := []struct {
		name       string
		beforeFunc func(t *testing.T, client *mocks.MockCognitoClient)
		giveAuth   CognitoAuth
		wantErr    error
	}{
		{
			name: "success",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient) {
				t.Helper()

				// Mock successful authentication response
				accessToken := "mock-access-token"
				refreshToken := "mock-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &accessToken,
						RefreshToken: &refreshToken,
						ExpiresIn:    3600,
					},
				}

				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return *input.ClientId == auth.AppClientID &&
						input.AuthFlow == types.AuthFlowTypeUserPasswordAuth &&
						input.AuthParameters["USERNAME"] == auth.Username &&
						input.AuthParameters["PASSWORD"] == auth.Password &&
						input.AuthParameters["SECRET_HASH"] != ""
				})).Return(mockOutput, nil)
			},
			giveAuth: auth,
			wantErr:  nil,
		},
		{
			name: "error",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient) {
				t.Helper()

				client.EXPECT().InitiateAuth(mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			giveAuth: auth,
			wantErr:  assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := mocks.NewMockCognitoClient(t)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, client)
			}

			source := newCognitoTokenSourceWithClient(tt.giveAuth, client)

			err := source.Authenticate(t.Context())
			assert.Equal(t, tt.wantErr, err)
		})
	}
}

func Test_CognitoTokenSource_Token(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *mocks.MockCognitoClient, *CognitoTokenSource)
		giveAuth   CognitoAuth
		wantToken  *oauth2.Token
		wantErr    error
	}{
		{
			name: "success with new result",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Mock successful authentication response
				accessToken := "new-access-token"
				refreshToken := "new-refresh-token"
				authResult := &types.AuthenticationResultType{
					AccessToken:  &accessToken,
					RefreshToken: &refreshToken,
					ExpiresIn:    3600,
				}
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: authResult,
				}

				client.EXPECT().InitiateAuth(mock.Anything, mock.Anything).Return(mockOutput, nil)
			},
			wantToken: &oauth2.Token{
				AccessToken: "new-access-token",
			},
		},
		{
			name: "success with cached result",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result to simulate cached token
				accessToken := "cached-access-token"
				refreshToken := "test-cached-refresh-token" //nolint:gosec // G101: Test value, not a real credential
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &accessToken,
					RefreshToken: &refreshToken,
				}
				// Set token expiry to future to avoid re-authentication
				source.tokenExpiry = time.Now().Add(time.Hour)
			},
			wantToken: &oauth2.Token{
				AccessToken: "cached-access-token",
			},
			wantErr: nil,
		},
		{
			name: "success with expired token - refresh",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result with expired token but valid refresh token
				expiredToken := "expired-access-token"
				refreshToken := "valid-refresh-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &expiredToken,
					RefreshToken: &refreshToken,
				}
				// Set token expiry to past to force refresh
				source.tokenExpiry = time.Now().Add(-time.Hour)

				// Mock successful refresh response using REFRESH_TOKEN_AUTH flow
				newAccessToken := "refreshed-access-token"
				newRefreshToken := "new-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}

				// Expect InitiateAuth call with REFRESH_TOKEN_AUTH flow
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeRefreshTokenAuth &&
						input.AuthParameters["REFRESH_TOKEN"] == refreshToken &&
						input.AuthParameters["SECRET_HASH"] != ""
				})).Return(mockOutput, nil)
			},
			wantToken: &oauth2.Token{
				AccessToken: "refreshed-access-token",
			},
			wantErr: nil,
		},
		{
			name: "expired token with refresh fallback to full auth",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result with expired token and refresh token
				expiredToken := "expired-access-token"
				expiredRefreshToken := "expired-refresh-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &expiredToken,
					RefreshToken: &expiredRefreshToken,
				}
				// Set token expiry to past to force refresh
				source.tokenExpiry = time.Now().Add(-time.Hour)

				// First call: refresh token fails
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeRefreshTokenAuth
				})).Return(nil, assert.AnError).Once()

				// Second call: fallback to full authentication succeeds
				newAccessToken := "new-access-token"
				newRefreshToken := "new-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeUserPasswordAuth
				})).Return(mockOutput, nil).Once()
			},
			wantToken: &oauth2.Token{
				AccessToken: "new-access-token",
			},
			wantErr: nil,
		},
		{
			name: "initiate auth error",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				client.EXPECT().InitiateAuth(mock.Anything, mock.Anything).Return(nil, assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := mocks.NewMockCognitoClient(t)
			auth := CognitoAuth{
				AWSRegion:       "us-east-1",
				AppClientID:     "test-client-id",
				AppClientSecret: "test-client-secret",
				Username:        "testuser",
				Password:        "testpass",
			}

			source := newCognitoTokenSourceWithClient(auth, client)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, client, source)
			}

			token, err := source.Token()

			if tt.wantErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.wantToken, token)
			}
		})
	}
}

func Test_CognitoTokenSource_RefreshToken(t *testing.T) {
	t.Parallel()

	auth := CognitoAuth{
		AWSRegion:       "us-east-1",
		AppClientID:     "test-client-id",
		AppClientSecret: "test-client-secret",
		Username:        "testuser",
		Password:        "testpass",
	}

	tests := []struct {
		name       string
		beforeFunc func(*testing.T, *mocks.MockCognitoClient, *CognitoTokenSource)
		wantErr    error
	}{
		{
			name: "success with valid refresh token",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result with existing tokens
				existingAccessToken := "existing-access-token"
				existingRefreshToken := "existing-refresh-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &existingAccessToken,
					RefreshToken: &existingRefreshToken,
				}
				source.tokenExpiry = time.Now().Add(-time.Hour) // Expired

				// Mock successful refresh response
				newAccessToken := "new-access-token"
				newRefreshToken := "new-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}

				// Expect InitiateAuth call with REFRESH_TOKEN_AUTH flow
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeRefreshTokenAuth &&
						input.AuthParameters["REFRESH_TOKEN"] == existingRefreshToken &&
						input.AuthParameters["SECRET_HASH"] != "" &&
						*input.ClientId == auth.AppClientID
				})).Return(mockOutput, nil)
			},
			wantErr: nil,
		},
		{
			name: "refresh token fails - fallback to full authentication",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result with existing tokens
				existingAccessToken := "existing-access-token"
				existingRefreshToken := "expired-refresh-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &existingAccessToken,
					RefreshToken: &existingRefreshToken,
				}

				// First call: refresh token fails
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeRefreshTokenAuth
				})).Return(nil, assert.AnError).Once()

				// Second call: fallback to full authentication succeeds
				newAccessToken := "fallback-access-token"
				newRefreshToken := "fallback-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeUserPasswordAuth
				})).Return(mockOutput, nil).Once()
			},
			wantErr: nil,
		},
		{
			name: "no refresh token available - fallback to full authentication",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result without refresh token
				existingAccessToken := "existing-access-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &existingAccessToken,
					RefreshToken: nil, // No refresh token
				}

				// Should call full authentication directly
				newAccessToken := "new-access-token"
				newRefreshToken := "new-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeUserPasswordAuth
				})).Return(mockOutput, nil)
			},
			wantErr: nil,
		},
		{
			name: "no auth result cached - fallback to full authentication",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// No auth result cached
				source.authResult = nil

				// Should call full authentication directly
				newAccessToken := "new-access-token"
				newRefreshToken := "new-refresh-token"
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken:  &newAccessToken,
						RefreshToken: &newRefreshToken,
						ExpiresIn:    3600,
					},
				}
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeUserPasswordAuth
				})).Return(mockOutput, nil)
			},
			wantErr: nil,
		},
		{
			name: "refresh fails and fallback also fails",
			beforeFunc: func(t *testing.T, client *mocks.MockCognitoClient, source *CognitoTokenSource) {
				t.Helper()

				// Pre-populate auth result with existing tokens
				existingAccessToken := "existing-access-token"
				existingRefreshToken := "expired-refresh-token"
				source.authResult = &types.AuthenticationResultType{
					AccessToken:  &existingAccessToken,
					RefreshToken: &existingRefreshToken,
				}

				// First call: refresh token fails
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeRefreshTokenAuth
				})).Return(nil, assert.AnError).Once()

				// Second call: fallback to full authentication also fails
				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return input.AuthFlow == types.AuthFlowTypeUserPasswordAuth
				})).Return(nil, assert.AnError).Once()
			},
			wantErr: assert.AnError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := mocks.NewMockCognitoClient(t)
			source := newCognitoTokenSourceWithClient(auth, client)

			if tt.beforeFunc != nil {
				tt.beforeFunc(t, client, source)
			}

			err := source.RefreshToken(context.Background())

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
				// Verify that the auth result and token expiry were updated
				assert.NotNil(t, source.authResult)
				assert.NotNil(t, source.authResult.AccessToken)
				assert.True(t, source.tokenExpiry.After(time.Now()), "Token expiry should be in the future")
			}
		})
	}
}

func Test_CognitoTokenSource_TokenExpiresAt(t *testing.T) {
	t.Parallel()

	expiresAt := time.Now().Add(time.Hour)

	source := CognitoTokenSource{
		tokenExpiry: expiresAt,
	}

	assert.Equal(t, expiresAt, source.TokenExpiresAt())
}

func Test_CognitoTokenSource_secretHash(t *testing.T) {
	t.Parallel()

	// Test with static values to ensure the hash calculation is correct
	auth := CognitoAuth{
		AppClientID:     "clientid",
		AppClientSecret: "secret",
		Username:        "user",
	}

	tokenSource := NewCognitoTokenSource(auth)
	hash := tokenSource.secretHash()

	// This is the expected base64-encoded HMAC-SHA256 hash for the above values
	// HMAC-SHA256("secret", "user" + "clientid") = base64 encoded result
	expected := "E1KXMtDWZqk4xodyW0dfVQzUoSoWg7hMk0yc2ermw4M="
	assert.Equal(t, expected, hash)
}
