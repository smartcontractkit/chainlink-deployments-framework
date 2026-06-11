package sui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	cslclient "github.com/smartcontractkit/chainlink-sui/relayer/client"
)

const defaultGrpcToken = "test"

// NewPTBClientFromNodeURL creates a gRPC-backed Sui PTB client from an HTTP RPC URL.
func NewPTBClientFromNodeURL(log logger.Logger, nodeURL string, grpcToken string) (cslclient.SuiPTBClient, error) {
	grpcTarget, err := grpcTargetFromNodeURL(nodeURL)
	if err != nil {
		return nil, err
	}
	if grpcToken == "" {
		grpcToken = defaultGrpcToken
	}

	return cslclient.NewPTBClient(log, cslclient.PTBClientConfig{
		GrpcTarget:            grpcTarget,
		GrpcToken:             grpcToken,
		TransactionTimeout:    30 * time.Second,
		MaxConcurrentRequests: 50,
		DefaultRequestType:    cslclient.WaitForEffectsCert,
	})
}

func grpcTargetFromNodeURL(nodeURL string) (string, error) {
	u, err := url.Parse(nodeURL)
	if err != nil {
		return "", fmt.Errorf("parse node URL %q: %w", nodeURL, err)
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		return "", fmt.Errorf("node URL %q has no host", nodeURL)
	}
	if port == "" {
		switch u.Scheme {
		case "https":
			port = "443"
		default:
			port = "9000"
		}
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		return fmt.Sprintf("[%s]:%s", host, port), nil
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}
