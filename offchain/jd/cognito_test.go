package jd

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"

	"github.com/smartcontractkit/chainlink-deployments-framework/offchain/jd/internal/mocks"
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
		AWSRegion:              "us-east-1",
		CognitoAppClientID:     "test-client-id",
		CognitoAppClientSecret: "test-client-secret",
		Username:               "testuser",
		Password:               "testpass",
	}

	tokenSource, err := NewCognitoTokenSource(auth)

	require.NoError(t, err)
	assert.NotNil(t, tokenSource)
	assert.Equal(t, auth, tokenSource.auth)
	assert.Nil(t, tokenSource.client)
	assert.Nil(t, tokenSource.authResult)
}

func Test_CognitoTokenSource_Authenticate(t *testing.T) {
	t.Parallel()

	auth := CognitoAuth{
		AWSRegion:              "us-east-1",
		CognitoAppClientID:     "test-client-id",
		CognitoAppClientSecret: "test-client-secret",
		Username:               "testuser",
		Password:               "testpass",
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
				mockOutput := &cognitoidentityprovider.InitiateAuthOutput{
					AuthenticationResult: &types.AuthenticationResultType{
						AccessToken: &accessToken,
					},
				}

				client.EXPECT().InitiateAuth(mock.Anything, mock.MatchedBy(func(input *cognitoidentityprovider.InitiateAuthInput) bool {
					return *input.ClientId == auth.CognitoAppClientID &&
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
				authResult := &types.AuthenticationResultType{
					AccessToken: &accessToken,
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
				source.authResult = &types.AuthenticationResultType{
					AccessToken: &accessToken,
				}

				client.AssertNotCalled(t, "InitiateAuth")
			},
			wantToken: &oauth2.Token{
				AccessToken: "cached-access-token",
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
				AWSRegion:              "us-east-1",
				CognitoAppClientID:     "test-client-id",
				CognitoAppClientSecret: "test-client-secret",
				Username:               "testuser",
				Password:               "testpass",
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

func Test_CognitoTokenSource_secretHash(t *testing.T) {
	t.Parallel()

	// Test with static values to ensure the hash calculation is correct
	auth := CognitoAuth{
		CognitoAppClientID:     "clientid",
		CognitoAppClientSecret: "secret",
		Username:               "user",
	}

	tokenSource, _ := NewCognitoTokenSource(auth)
	hash := tokenSource.secretHash()

	// This is the expected base64-encoded HMAC-SHA256 hash for the above values
	// HMAC-SHA256("secret", "user" + "clientid") = base64 encoded result
	expected := "E1KXMtDWZqk4xodyW0dfVQzUoSoWg7hMk0yc2ermw4M="
	assert.Equal(t, expected, hash)
}
