// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newGraphQLServer returns a test server that records the headers of the last
// GraphQL request and replies with the given raw JSON body.
func newGraphQLServer(t *testing.T, reply string, gotHeaders *http.Header) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotHeaders != nil {
			*gotHeaders = r.Header.Clone()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(reply))
	}))
}

func TestNew_validation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cfg  Config
	}{
		{"missing org", Config{Token: "t"}},
		{"no credentials", Config{OrganizationID: "org"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := New(context.Background(), tc.cfg); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestExecute_staticToken_sendsHeaders(t *testing.T) {
	t.Parallel()
	var got http.Header
	srv := newGraphQLServer(t, `{"data":{"getMyAccount":{"id":"acc_1"}}}`, &got)
	defer srv.Close()

	c, err := New(context.Background(), Config{
		Endpoint:       srv.URL,
		OrganizationID: "org-short-id",
		Token:          "static-token",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var out struct {
		GetMyAccount struct {
			ID string `json:"id"`
		} `json:"getMyAccount"`
	}
	if err := c.Execute(context.Background(), `query { getMyAccount { id } }`, nil, &out); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if want := "Bearer static-token"; got.Get("Authorization") != want {
		t.Errorf("Authorization = %q, want %q", got.Get("Authorization"), want)
	}
	if want := "org-short-id"; got.Get("organization-id") != want {
		t.Errorf("organization-id = %q, want %q", got.Get("organization-id"), want)
	}
	if out.GetMyAccount.ID != "acc_1" {
		t.Errorf("data not unmarshaled: got %+v", out)
	}
}

func TestExecute_applicationFlow_exchangesToken(t *testing.T) {
	t.Parallel()

	// Fake OAuth2 token endpoint for the client-credentials grant.
	var tokenForm url.Values
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		tokenForm = r.Form
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"service-token","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	var got http.Header
	gqlSrv := newGraphQLServer(t, `{"data":{}}`, &got)
	defer gqlSrv.Close()

	c, err := New(context.Background(), Config{
		Endpoint:       gqlSrv.URL,
		OrganizationID: "org",
		ClientID:       "cid",
		ClientSecret:   "secret",
		TokenURL:       tokenSrv.URL,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := c.Execute(context.Background(), `query { __typename }`, nil, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if tokenForm.Get("grant_type") != "client_credentials" {
		t.Errorf("grant_type = %q, want client_credentials", tokenForm.Get("grant_type"))
	}
	if want := "Bearer service-token"; got.Get("Authorization") != want {
		t.Errorf("Authorization = %q, want %q", got.Get("Authorization"), want)
	}
	if got.Get("organization-id") != "org" {
		t.Errorf("organization-id = %q, want org", got.Get("organization-id"))
	}
}

func TestExecute_graphQLErrors(t *testing.T) {
	t.Parallel()
	srv := newGraphQLServer(t, `{"errors":[{"message":"not authorized"},{"message":"bad field"}]}`, nil)
	defer srv.Close()

	c, err := New(context.Background(), Config{
		Endpoint:       srv.URL,
		OrganizationID: "org",
		Token:          "t",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	err = c.Execute(context.Background(), `query { x }`, nil, nil)
	if err == nil {
		t.Fatal("expected error from graphql errors, got nil")
	}
	if !strings.Contains(err.Error(), "not authorized") || !strings.Contains(err.Error(), "bad field") {
		t.Errorf("error %q does not contain both messages", err.Error())
	}
}

func TestExecute_sendsQueryAndVariables(t *testing.T) {
	t.Parallel()
	var body request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	c, err := New(context.Background(), Config{
		Endpoint:       srv.URL,
		OrganizationID: "org",
		Token:          "t",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	vars := map[string]any{"id": "abc"}
	if err := c.Execute(context.Background(), `query($id:ID!){ node(id:$id){ id } }`, vars, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(body.Query, "node(id:$id)") {
		t.Errorf("query not forwarded: %q", body.Query)
	}
	if body.Variables["id"] != "abc" {
		t.Errorf("variables not forwarded: %+v", body.Variables)
	}
}
