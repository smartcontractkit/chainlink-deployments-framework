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
//
// The gRPC auth token is resolved in priority order:
//  1. The grpcToken argument, when non-empty. This is the primary path: the deployment framework
//     threads cfgnet.RPC.AuthToken through RPCChainProviderConfig into this argument, so an
//     authenticated endpoint such as Alchemy is configured via a first-class token field rather
//     than embedded in the URL.
//  2. The URL userinfo (e.g. https://<token>@host), used only as a legacy fallback when grpcToken
//     is empty but the URL carries userinfo. grpcTargetFromNodeURL strips userinfo from the gRPC
//     target, so only the host:port is dialed regardless.
//  3. defaultGrpcToken ("test"), preserving prior behavior for unauthenticated endpoints (local
//     nodes, public fullnodes).
//
// The resolved token is sent as gRPC metadata (Bearer / x-api-key / x-token / x-spectrum-auth) by
// suigrpcconn.authMetadata. Note: it is applied to the gRPC transport only; the JSON-RPC/devInspect
// client built downstream from the bare gRPC target does not receive it.
func NewPTBClientFromNodeURL(log logger.Logger, nodeURL string, grpcToken string) (cslclient.SuiPTBClient, error) {
	grpcTarget, err := grpcTargetFromNodeURL(nodeURL)
	if err != nil {
		return nil, err
	}
	grpcToken = grpcTokenFromNodeURL(nodeURL, grpcToken)

	return cslclient.NewPTBClient(log, cslclient.PTBClientConfig{
		GrpcTarget:            grpcTarget,
		GrpcToken:             grpcToken,
		TransactionTimeout:    30 * time.Second,
		MaxConcurrentRequests: 50,
		DefaultRequestType:    cslclient.WaitForEffectsCert,
	})
}

// grpcTokenFromNodeURL resolves the gRPC auth token in the priority order documented on
// NewPTBClientFromNodeURL: the explicit token arg when non-empty, then the URL userinfo
// username (e.g. https://<token>@host) as a legacy fallback, then defaultGrpcToken. The URL
// is re-parsed for userinfo only when no explicit token is supplied; a parse error is ignored
// and falls through to the default, matching prior behavior.
func grpcTokenFromNodeURL(nodeURL, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if u, err := url.Parse(nodeURL); err == nil && u.User != nil {
		if user := u.User.Username(); user != "" {
			return user
		}
	}
	return defaultGrpcToken
}

func grpcTargetFromNodeURL(nodeURL string) (string, error) {
	u, err := url.Parse(nodeURL)
	if err != nil {
		return "", fmt.Errorf("parse node URL %q: %w", redactURL(nodeURL), err)
	}
	host := u.Hostname()
	port := u.Port()
	if host == "" {
		return "", fmt.Errorf("node URL %q has no host", redactURL(nodeURL))
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

// redactURL returns a log-safe representation of a node URL with any userinfo removed, so that
// error messages never echo the raw URL and leak an auth token into logs. In the legacy
// https://<token>@host form the token is the userinfo *username*, so url.URL.Redacted (which
// only masks the password) is insufficient; the entire u.User is cleared instead. When the URL
// cannot be parsed at all, userinfo is redacted best-effort by dropping everything before the
// last '@'; if there is no '@' the raw input is returned as-is since it carries no userinfo.
func redactURL(raw string) string {
	if u, err := url.Parse(raw); err == nil {
		u.User = nil
		return u.String()
	}
	if i := strings.LastIndex(raw, "@"); i >= 0 {
		return "<redacted>@" + raw[i+1:]
	}
	return raw
}
