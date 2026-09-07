package handler

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/anthropic/oidc-platform/internal/domain"
)

func TestDiscordGuildConfigRoundTrip(t *testing.T) {
	var req updateProviderRequest
	if err := json.Unmarshal([]byte(`{"guild_ids":["123456789012345678","123456789012345678"," 987654321098765432 "]}`), &req); err != nil {
		t.Fatal(err)
	}
	pc := &domain.ProviderConfig{Provider: domain.ProviderDiscord}
	applyProviderRequest(pc, req)
	want := []string{"123456789012345678", "987654321098765432"}
	stored, err := json.Marshal(pc.ExtraConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stored, &pc.ExtraConfig); err != nil {
		t.Fatal(err)
	}
	if got := providerPayload(pc)["guild_ids"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	applyProviderRequest(pc, updateProviderRequest{})
	if !reflect.DeepEqual(pc.DiscordGuildIDs(), want) {
		t.Fatal("omitted guild_ids should preserve configuration")
	}
	applyProviderRequest(pc, updateProviderRequest{GuildIDs: []string{}})
	if len(pc.DiscordGuildIDs()) != 0 {
		t.Fatal("empty guild_ids should clear configuration")
	}
	if err := json.Unmarshal([]byte(`{"guild_ids":[123456789012345678]}`), &req); err == nil {
		t.Fatal("numeric IDs must not be accepted with precision loss")
	}
}
