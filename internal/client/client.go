// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

// Package client provides a thin, authenticated GraphQL client for the
// WellPlayed API (https://well-played.gg/).
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// DefaultEndpoint is the production WellPlayed GraphQL endpoint.
	DefaultEndpoint = "https://api.warrior.well-played.gg/graphql"

	// DefaultTokenURL is the production OAuth2 token endpoint used by the
	// application (client-credentials) flow.
	DefaultTokenURL = "https://oauth.warrior.well-played.gg/oauth2/token"
)

// Config holds everything needed to build an authenticated Client.
//
// Exactly one authentication flow must be configured:
//   - Application flow: set ClientID and ClientSecret. The client exchanges
//     them for a service token at TokenURL using the client-credentials grant
//     and refreshes it automatically.
//   - Static token flow: set Token to a pre-obtained OIDC access token. No
//     refresh is performed; the caller owns the token lifecycle.
type Config struct {
	// Endpoint is the GraphQL endpoint. Defaults to DefaultEndpoint.
	Endpoint string
	// OrganizationID is the org short id sent in the organization-id header.
	OrganizationID string

	// Token is a pre-obtained OIDC access token (static token flow).
	Token string

	// ClientID and ClientSecret enable the application flow.
	ClientID     string
	ClientSecret string
	// TokenURL is the OAuth2 token endpoint. Defaults to DefaultTokenURL.
	TokenURL string
	// Scopes optionally requested during the client-credentials exchange.
	Scopes []string
}

// Client is an authenticated GraphQL client. It is safe for concurrent use.
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// New builds a Client from cfg, wiring up the requested authentication flow.
// The returned client injects the authorization and organization-id headers
// on every request.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.OrganizationID == "" {
		return nil, fmt.Errorf("organization id is required")
	}

	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}

	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}

	// The oauth2 helpers capture the context they're given and reuse it for
	// every future (lazy) token fetch. This Client outlives the request-scoped
	// context handed to New (e.g. Terraform cancels its Configure context as
	// soon as Configure returns, long before the first token is fetched during
	// Apply). Use a background context so token refreshes aren't aborted with
	// "context canceled". Per-request cancellation still applies via the
	// context passed to Execute.
	authCtx := context.Background()

	var base *http.Client
	switch {
	case cfg.Token != "":
		// Static token flow: no refresh, caller owns the lifecycle.
		src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.Token})
		base = oauth2.NewClient(authCtx, src)
	case cfg.ClientID != "" && cfg.ClientSecret != "":
		// Application flow: client-credentials grant with auto-refresh.
		ccCfg := clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     tokenURL,
			Scopes:       cfg.Scopes,
			// The WellPlayed OAuth client uses token_endpoint_auth_method
			// 'client_secret_post': credentials go in the request body, not an
			// HTTP Basic auth header.
			AuthStyle: oauth2.AuthStyleInParams,
		}
		base = ccCfg.Client(authCtx)
	default:
		return nil, fmt.Errorf("no credentials configured: set either token, or both client_id and client_secret")
	}

	// Wrap the auth transport so every request also carries organization-id.
	base.Transport = &headerTransport{
		base:    base.Transport,
		headers: map[string]string{"organization-id": cfg.OrganizationID},
	}

	return &Client{httpClient: base, endpoint: endpoint}, nil
}

// headerTransport injects static headers onto every outgoing request.
type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	// Clone to avoid mutating a request the caller may reuse.
	req = req.Clone(req.Context())
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	return base.RoundTrip(req)
}

// request is the JSON body of a GraphQL POST.
type request struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// gqlError mirrors a single entry of the GraphQL errors array.
type gqlError struct {
	Message string `json:"message"`
}

func (e gqlError) Error() string { return e.Message }

// response is the JSON envelope returned by the GraphQL endpoint.
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors []gqlError      `json:"errors"`
}

// Execute runs a GraphQL query/mutation. If out is non-nil, the contents of
// the response "data" field are unmarshaled into it. GraphQL-level errors are
// returned joined into a single error.
func (c *Client) Execute(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(request{Query: query, Variables: variables})
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %s: %s", httpResp.Status, strings.TrimSpace(string(raw)))
	}

	var resp response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if len(resp.Errors) > 0 {
		msgs := make([]string, len(resp.Errors))
		for i, e := range resp.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("graphql error: %s", strings.Join(msgs, "; "))
	}

	if out != nil {
		if err := json.Unmarshal(resp.Data, out); err != nil {
			return fmt.Errorf("unmarshaling data: %w", err)
		}
	}

	return nil
}
