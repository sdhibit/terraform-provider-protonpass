// Copyright (c) PlaneOpsCc
// SPDX-License-Identifier: MPL-2.0

package passcli_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/planeopscc/terraform-provider-protonpass/internal/passcli"
	"github.com/planeopscc/terraform-provider-protonpass/internal/testutil"
)

// TestAcc_HealthCheck verifies the session probe against the live CLI. It is
// the regression guard for pass-cli 2.2.4 removing `test`: a provider that
// still probed with `test` would fail configuration here.
func TestAcc_HealthCheck(t *testing.T) {
	testutil.SkipIfNotAcc(t)
	client := testutil.NewAccClient(t)

	if err := client.HealthCheck(t.Context()); err != nil {
		t.Fatalf("HealthCheck against live pass-cli failed: %v — run 'pass-cli login' first", err)
	}
}

// TestAcc_ListItemsPopulatesTitleAndType is the regression guard for the
// `item list` shape change in pass-cli 2.0.3. Listing is read-only, so this
// runs against whatever vaults the live session already has. Empty titles or
// every item typed "note" means the summary shape is not being parsed.
func TestAcc_ListItemsPopulatesTitleAndType(t *testing.T) {
	testutil.SkipIfNotAcc(t)
	client := testutil.NewAccClient(t)
	ctx := t.Context()

	if err := client.HealthCheck(ctx); err != nil {
		t.Skipf("pass-cli session not active: %v — run 'pass-cli login' first", err)
	}

	vaults, err := client.ListVaults(ctx)
	if err != nil {
		t.Fatalf("ListVaults: %v", err)
	}
	if len(vaults) == 0 {
		t.Skip("no vaults in this account")
	}

	var inspected int
	for _, vault := range vaults {
		items, err := client.ListItemsInVault(ctx, vault.ShareID)
		if err != nil {
			t.Fatalf("ListItemsInVault(%q): %v", vault.ShareID, err)
		}
		for _, item := range items {
			inspected++
			if item.ItemID == "" {
				t.Errorf("vault %q: item has empty ItemID", vault.Name)
			}
			// Titles are mandatory in Proton Pass, so an empty one here means
			// the response was parsed against the wrong shape.
			if item.Title == "" {
				t.Errorf("vault %q: item %q has empty Title — `item list` shape not parsed", vault.Name, item.ItemID)
			}
			if item.Type == "" {
				t.Errorf("vault %q: item %q has empty Type", vault.Name, item.ItemID)
			}
		}
	}

	if inspected == 0 {
		t.Skip("no items in any vault")
	}
	t.Logf("inspected %d items across %d vaults", inspected, len(vaults))
}

// TestAcc_ItemCreateRoundTrip is the regression guard for item creation. Under
// the old title-lookup readback, every one of these creates succeeded in Proton
// Pass and then reported "item not found after creation", failing the apply and
// leaving the item orphaned. It works in a throwaway vault it deletes on exit.
func TestAcc_ItemCreateRoundTrip(t *testing.T) {
	testutil.SkipIfNotAcc(t)
	client := testutil.NewAccClient(t)
	ctx := t.Context()

	if err := client.HealthCheck(ctx); err != nil {
		t.Skipf("pass-cli session not active: %v — run 'pass-cli login' first", err)
	}

	vaultName := fmt.Sprintf("tf-acc-compat-%d", time.Now().UnixMilli())
	vault, err := client.CreateVault(ctx, vaultName)
	if err != nil {
		t.Fatalf("CreateVault(%q): %v", vaultName, err)
	}
	t.Cleanup(func() {
		if err := client.DeleteVault(context.Background(), vault.ShareID); err != nil {
			t.Errorf("cleanup: DeleteVault(%q) failed, remove it manually: %v", vault.ShareID, err)
		}
	})

	tests := []struct {
		name   string
		title  string
		typ    string
		create func(title string) (*passcli.ItemJSON, error)
	}{
		{
			name: "login", title: "compat-login", typ: "login",
			create: func(title string) (*passcli.ItemJSON, error) {
				return client.CreateItemLogin(ctx, vault.ShareID, title, "alice", "pw-123", "", []string{"https://example.com"})
			},
		},
		{
			name: "note", title: "compat-note", typ: "note",
			create: func(title string) (*passcli.ItemJSON, error) {
				return client.CreateItemNote(ctx, vault.ShareID, title, "note body")
			},
		},
		{
			name: "credit-card", title: "compat-cc", typ: "credit-card",
			create: func(title string) (*passcli.ItemJSON, error) {
				return client.CreateItemCreditCard(ctx, vault.ShareID, title, "John Doe", "4111111111111111", "123", "2030-01", "1234")
			},
		},
		{
			name: "wifi", title: "compat-wifi", typ: "wifi",
			create: func(title string) (*passcli.ItemJSON, error) {
				return client.CreateItemWiFi(ctx, vault.ShareID, title, "test-ssid", "wifi-pw", "WPA2")
			},
		},
		{
			name: "ssh-key", title: "compat-ssh", typ: "ssh-key",
			create: func(title string) (*passcli.ItemJSON, error) {
				return client.CreateItemSSHKey(ctx, vault.ShareID, title, "ed25519", "compat test")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item, err := tc.create(tc.title)
			if err != nil {
				t.Fatalf("create %s: %v", tc.name, err)
			}
			if item.ItemID == "" {
				t.Fatalf("create %s: returned no item ID", tc.name)
			}
			if item.Title != tc.title {
				t.Errorf("create %s: Title = %q, want %q", tc.name, item.Title, tc.title)
			}
			if item.Type != tc.typ {
				t.Errorf("create %s: Type = %q, want %q", tc.name, item.Type, tc.typ)
			}

			// The item must also be findable by the listing path.
			items, err := client.ListItemsInVault(ctx, vault.ShareID)
			if err != nil {
				t.Fatalf("ListItemsInVault: %v", err)
			}
			var found bool
			for _, listed := range items {
				if listed.ItemID == item.ItemID {
					found = true
					if listed.Title != tc.title {
						t.Errorf("listed %s: Title = %q, want %q", tc.name, listed.Title, tc.title)
					}
					if listed.Type != tc.typ {
						t.Errorf("listed %s: Type = %q, want %q", tc.name, listed.Type, tc.typ)
					}
				}
			}
			if !found {
				t.Errorf("created %s item %q not present in `item list`", tc.name, item.ItemID)
			}
		})
	}
}
