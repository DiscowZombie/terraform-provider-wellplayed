// Copyright (c) Mathéo Cimbaro
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is used to instantiate a provider during
// acceptance testing. The factory function is called for each Terraform CLI
// command to create a provider server that the CLI can connect to.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"wellplayed": providerserver.NewProtocol6WithError(New("test")()),
}

// Acceptance tests configure the provider entirely from the environment, and
// need a real organization plus a user and two groups to move a member
// between. These are deployment-specific, so they come from env vars.
const (
	envTestUserID = "WELLPLAYED_TEST_USER_ID"
	envTestGroup  = "WELLPLAYED_TEST_GROUP_ID"
	envTestGroup2 = "WELLPLAYED_TEST_GROUP_ID_2"
)

// testAccPreCheckProvider asserts only the provider-level requirements: an
// organization id plus one configured auth flow. Use it for resources that
// don't need the IAM-specific user/group fixtures.
func testAccPreCheckProvider(t *testing.T) {
	t.Helper()

	if os.Getenv(envOrganizationID) == "" {
		t.Fatalf("%s must be set for acceptance tests", envOrganizationID)
	}

	hasToken := os.Getenv(envToken) != ""
	hasApp := os.Getenv(envClientID) != "" && os.Getenv(envClientSecret) != ""
	if !hasToken && !hasApp {
		t.Fatalf("set %s, or both %s and %s, for acceptance tests", envToken, envClientID, envClientSecret)
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()

	for _, k := range []string{envTestUserID, envTestGroup, envTestGroup2} {
		if os.Getenv(k) == "" {
			t.Fatalf("%s must be set for acceptance tests", k)
		}
	}
	testAccPreCheckProvider(t)
}
