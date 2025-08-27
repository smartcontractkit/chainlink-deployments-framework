package jd

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"

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

	// The Cognito client interface for making API calls
	client CognitoClient
}

// CognitoAuth contains the Cognito authentication information required to generate a token from
// Cognito.
type CognitoAuth struct {
	AWSRegion              string
	CognitoAppClientID     string
	CognitoAppClientSecret string
	Username               string
	Password               string
}

// NewCognitoTokenSource creates a new CognitoTokenSource with the given CognitoAuth configuration.
// If client is nil, a real AWS Cognito client will be created when Authenticate is called.
func NewCognitoTokenSource(auth CognitoAuth) (*CognitoTokenSource, error) {
	return &CognitoTokenSource{
		auth: auth,
	}, nil
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
		ClientId: aws.String(c.auth.CognitoAppClientID),
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

	c.authResult = output.AuthenticationResult

	return nil
}

// Token retrieves an OAuth2 access token for authenticating with the Job Distributor service.
//
// This method implements a lazy loading pattern:
//  1. If an authentication result is already cached, it returns the cached access token
//  2. If no cached result exists, it automatically authenticates with AWS Cognito first
//  3. Returns the access token wrapped in an oauth2.Token struct
//
// The method implements the oauth2.TokenSource interface, making it compatible with
// standard OAuth2 client libraries and HTTP clients that support token sources.
//
// Note: This method uses context.Background() for authentication if no cached token exists.
// For more control over authentication context and timeout behavior, consider calling
// Authenticate() explicitly before calling Token().
//
// Returns an OAuth2 token containing the Cognito access token
func (c *CognitoTokenSource) Token() (*oauth2.Token, error) {
	if c.authResult == nil {
		if err := c.Authenticate(context.Background()); err != nil {
			return nil, err
		}
	}

	return &oauth2.Token{
		AccessToken: aws.ToString(c.authResult.AccessToken),
	}, nil
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
	hmac := hmac.New(sha256.New, []byte(c.auth.CognitoAppClientSecret))
	message := []byte(c.auth.Username + c.auth.CognitoAppClientID)
	hmac.Write(message)
	dataHmac := hmac.Sum(nil)

	return base64.StdEncoding.EncodeToString(dataHmac)
}
