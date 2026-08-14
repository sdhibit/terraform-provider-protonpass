// Copyright (c) PlaneOpsCc
// SPDX-License-Identifier: MPL-2.0

package passcli_test

import (
	"testing"

	"github.com/planeopscc/terraform-provider-protonpass/internal/passcli"
	"github.com/planeopscc/terraform-provider-protonpass/internal/testutil"
)

// pass-cli >= 2.0.3 strips `content` from plain `item list` output so that
// listing can never leak secret material, promoting title and item_type to the
// top level instead. Parsing must survive that, otherwise every listed item
// comes back with an empty title and is misreported as a note.
func TestListItemsInVault_SummaryShape(t *testing.T) {
	fixture := loadFixture(t, "item_list_summary_v2.json")
	runner := testutil.NewFakeRunner(map[string]testutil.FakeResponse{
		"item list": {Stdout: fixture},
	})
	client := passcli.NewClient(runner)

	items, err := client.ListItemsInVault(t.Context(), "share-abc-123")
	if err != nil {
		t.Fatalf("ListItemsInVault returned error: %v", err)
	}
	if len(items) != 7 {
		t.Fatalf("expected 7 items, got %d", len(items))
	}

	want := []struct {
		itemID string
		title  string
		typ    string
	}{
		{"item-login-001", "Database Credentials", "login"},
		{"item-note-001", "My Note", "note"},
		{"item-cc-002", "My Visa", "credit-card"},
		{"item-wifi-003", "Office WiFi", "wifi"},
		{"item-ssh-004", "Deploy Key", "ssh-key"},
		{"item-id-005", "My Identity", "identity"},
		{"item-alias-006", "Newsletter Alias", "alias"},
	}
	for i, w := range want {
		got := items[i]
		if got.ItemID != w.itemID {
			t.Errorf("item %d: ItemID = %q, want %q", i, got.ItemID, w.itemID)
		}
		if got.Title != w.title {
			t.Errorf("item %d: Title = %q, want %q", i, got.Title, w.title)
		}
		if got.Type != w.typ {
			t.Errorf("item %d (%s): Type = %q, want %q", i, w.title, got.Type, w.typ)
		}
		if got.ShareID != "share-abc-123" {
			t.Errorf("item %d: ShareID = %q, want %q", i, got.ShareID, "share-abc-123")
		}
	}
}

// The nested shape is still what `item view` and `item list --show-secrets`
// return, and what pass-cli < 2.0.3 returns from plain `item list`.
func TestFlattenItem_NestedShapeStillWins(t *testing.T) {
	raw := passcli.ItemRawJSON{
		ID:      "item-1",
		ShareID: "share-1",
		Content: passcli.ItemContentJSON{
			Title: "Nested Title",
			Content: passcli.ItemTypedContent{
				Login: &passcli.LoginContentJSON{Username: "alice", Password: "s3cret"},
			},
		},
		// A CLI that emits both must not have the summary override the real content.
		SummaryTitle:    "Summary Title",
		SummaryItemType: "note",
	}

	item := passcli.FlattenItem(raw)
	if item.Title != "Nested Title" {
		t.Errorf("Title = %q, want %q", item.Title, "Nested Title")
	}
	if item.Type != "login" {
		t.Errorf("Type = %q, want %q", item.Type, "login")
	}
	if item.Username != "alice" {
		t.Errorf("Username = %q, want %q", item.Username, "alice")
	}
}

// With neither nested content nor an item_type, the historical assumption that
// a contentless item is a note must still hold.
func TestFlattenItem_NoTypeSignalDefaultsToNote(t *testing.T) {
	raw := passcli.ItemRawJSON{
		ID:      "item-1",
		Content: passcli.ItemContentJSON{Title: "Some Note", Note: "body"},
	}
	item := passcli.FlattenItem(raw)
	if item.Type != "note" {
		t.Errorf("Type = %q, want %q", item.Type, "note")
	}
	if item.Note != "body" {
		t.Errorf("Note = %q, want %q", item.Note, "body")
	}
}

// An item_type this provider does not model must not be silently relabelled as
// a note; falling through to "note" would let the items data source report a
// wrong type rather than an unknown one.
func TestFlattenItem_UnknownSummaryTypeFallsBackToNote(t *testing.T) {
	raw := passcli.ItemRawJSON{
		ID:              "item-1",
		SummaryTitle:    "Mystery",
		SummaryItemType: "something_new",
	}
	item := passcli.FlattenItem(raw)
	if item.Title != "Mystery" {
		t.Errorf("Title = %q, want %q", item.Title, "Mystery")
	}
	if item.Type != "note" {
		t.Errorf("Type = %q, want %q", item.Type, "note")
	}
}
