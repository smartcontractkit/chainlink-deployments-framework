package jd

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"golang.org/x/oauth2"
)

// CognitoClient defines the interface for Cognito Identity Provider operations.
// This interface allows for mocking the AWS Cognito client in unit tests.
type CognitoClient interface {
	InitiateAuth(
		ctx context.Context,
		params *cognitoidentityprovider.InitiateAuthInput,
		optFns ...func(*cognitoidentityprovider.Options),
	) (*cognitoidentityprovider.InitiateAuthOutput, error)
}

// CognitoTokenSource provides a oath2 token that is used to authenticate with the Job Distributor.
type CognitoTokenSource struct {
	// The Cognito authentication information
	auth CognitoAuth

	// The cached authentication result from Cognito
	authResult *types.AuthenticationResultType

	// The time when the cached token expires
	tokenExpiry time.Time

	// The Cognito client interface for making API calls
	client CognitoClient
}

// CognitoAuth contains the Cognito authentication information required to generate a token from
// Cognito.
type CognitoAuth struct {
	AWSRegion       string
	AppClientID     string
	AppClientSecret string
	Username        string
	Password        string
}

// NewCognitoTokenSource creates a new CognitoTokenSource with the given CognitoAuth configuration.
// If client is nil, a real AWS Cognito client will be created when Authenticate is called.
func NewCognitoTokenSource(auth CognitoAuth) *CognitoTokenSource {
	return &CognitoTokenSource{
		auth: auth,
	}
}

// Authenticate performs user authentication against AWS Cognito using the USER_PASSWORD_AUTH flow.
//
// Authentication results are cached in the authResult field and used by the Token() method
// to provide OAuth2 access tokens without re-authenticating.
func (c *CognitoTokenSource) Authenticate(ctx context.Context) error {
	// Create client if not already set (for production use)
	if c.client == nil {
		sdkConfig, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(c.auth.AWSRegion),
		)
		if err != nil {
			return err
		}
		c.client = cognitoidentityprovider.NewFromConfig(sdkConfig)
	}

	// Authenticate the user
	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeUserPasswordAuth,
		ClientId: aws.String(c.auth.AppClientID),
		AuthParameters: map[string]string{
			"USERNAME":    c.auth.Username,
			"PASSWORD":    c.auth.Password,
			"SECRET_HASH": c.secretHash(),
		},
	}

	output, err := c.client.InitiateAuth(ctx, input)
	if err != nil {
		return err
	}

	c.setAuthResult(output.AuthenticationResult)

	return nil
}

// Token retrieves an OAuth2 access token for authenticating with the Job Distributor service.
//
// This method implements a lazy loading pattern with automatic token refresh:
//  1. If no authentication result is cached, it authenticates with AWS Cognito
//  2. If the cached token has expired, it refreshes using the refresh token (REFRESH_TOKEN_AUTH flow)
//  3. Otherwise, it returns the cached access token
//  4. Returns the access token wrapped in an oauth2.Token struct
//
// The method implements the oauth2.TokenSource interface, making it compatible with
// standard OAuth2 client libraries and HTTP clients that support token sources. Following OAuth2
// best practices, it uses the refresh token when available to refresh the access token.
//
// Note: This method uses context.Background() for authentication/refresh if no cached token exists
// or if the token needs to be refreshed. For more control over authentication context and
// timeout behavior, consider calling Authenticate() or RefreshToken() explicitly before calling
// Token().
//
// Returns an OAuth2 token containing the Cognito access token
func (c *CognitoTokenSource) Token() (*oauth2.Token, error) {
	ctx := context.Background()

	// Check if we need to authenticate (no token or token expired)
	if c.authResult == nil {
		// No token cached, perform full authentication
		if err := c.Authenticate(ctx); err != nil {
			return nil, err
		}
	}

	// Check if the token has expired and refresh if necessary
	if time.Now().After(c.tokenExpiry) {
		// Token expired, try to refresh using refresh token
		if err := c.RefreshToken(ctx); err != nil {
			return nil, err
		}
	}

	return &oauth2.Token{
		AccessToken: aws.ToString(c.authResult.AccessToken),
	}, nil
}

// RefreshToken refreshes the access token using the stored refresh token via the REFRESH_TOKEN_AUTH flow.
//
// This method uses the refresh token from the cached authentication result to obtain new access and ID tokens
// without reusing the user's credentials. This is more efficient than full re-authentication
// and follows OAuth2 best practices.
//
// The method:
//  1. Uses the cached refresh token to call InitiateAuth with REFRESH_TOKEN_AUTH flow
//  2. Updates the cached authentication result with new tokens
//  3. Calculates and stores the new token expiry time
//
// Returns an error if the refresh fails (e.g., refresh token expired or invalid).
func (c *CognitoTokenSource) RefreshToken(ctx context.Context) error {
	if c.authResult == nil || c.authResult.RefreshToken == nil {
		return c.Authenticate(ctx) // Fall back to full authentication if no refresh token
	}

	input := &cognitoidentityprovider.InitiateAuthInput{
		AuthFlow: types.AuthFlowTypeRefreshTokenAuth,
		ClientId: aws.String(c.auth.AppClientID),
		AuthParameters: map[string]string{
			"REFRESH_TOKEN": aws.ToString(c.authResult.RefreshToken),
			"SECRET_HASH":   c.secretHash(),
		},
	}

	output, err := c.client.InitiateAuth(ctx, input)
	if err != nil {
		// If refresh fails, fall back to full authentication
		return c.Authenticate(ctx)
	}

	// Set the new authentication result and token expiry time
	c.setAuthResult(output.AuthenticationResult)

	return nil
}

// TokenExpiresAt returns the time when the cached token expires.
func (c *CognitoTokenSource) TokenExpiresAt() time.Time {
	return c.tokenExpiry
}

// secretHash computes the AWS Cognito secret hash required for authentication with app clients that have a client secret.
//
// The secret hash is calculated using HMAC-SHA256 with the following formula:
//
//	HMAC-SHA256(ClientSecret, Username + ClientId)
//
// This method:
//  1. Creates an HMAC-SHA256 hash using the Cognito app client secret as the key
//  2. Constructs the message by concatenating the username and client ID
//  3. Computes the HMAC hash of the message
//  4. Returns the hash encoded as a base64 string
//
// The secret hash is required by AWS Cognito when the app client is configured with a client secret.
// It provides an additional layer of security by ensuring that only clients with the correct secret
// can authenticate users.
//
// Returns the computed secret hash as a base64-encoded string.
func (c *CognitoTokenSource) secretHash() string {
	hmac := hmac.New(sha256.New, []byte(c.auth.AppClientSecret))
	message := []byte(c.auth.Username + c.auth.AppClientID)
	hmac.Write(message)
	dataHmac := hmac.Sum(nil)

	return base64.StdEncoding.EncodeToString(dataHmac)
}

// setAuthResult sets the authentication result and token expiry time
func (c *CognitoTokenSource) setAuthResult(authResult *types.AuthenticationResultType) {
	// Set the new authentication result
	c.authResult = authResult

	// Calculate and set the token expiry by appending expiresIn (seconds) to the current time
	expiresDuration := time.Duration(authResult.ExpiresIn) * time.Second
	c.tokenExpiry = time.Now().Add(expiresDuration)
}
