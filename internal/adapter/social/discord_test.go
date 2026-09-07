package social

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"golang.org/x/oauth2"
)

type discordTestTransport func(*http.Request) (*http.Response, error)

func (f discordTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func discordTestResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}

func TestDiscordGuildSnapshot(t *testing.T) {
	const guildID = "123456789012345678"
	cases := []struct {
		name                  string
		status                int
		body                  string
		known, member, joined bool
	}{
		{"member", 200, `{"user":{"id":"987654321098765432"},"joined_at":"2025-01-02T03:04:05.123456+00:00"}`, true, true, true},
		{"not a member", 404, `{"code":10007}`, true, false, false},
		{"unknown guild", 404, `{"code":10004}`, true, false, false},
		{"unrecognized 404", 404, `{}`, false, false, false},
		{"missing scope", 403, `{"code":50001}`, false, false, false},
		{"expired token", 401, `{}`, false, false, false},
		{"rate limited", 429, `{"retry_after":30}`, false, false, false},
		{"server failure", 500, `{}`, false, false, false},
		{"invalid json", 200, `{`, false, false, false},
		{"empty member", 200, `null`, false, false, false},
		{"wrong user", 200, `{"user":{"id":"111"},"joined_at":"2025-01-01T00:00:00Z"}`, false, false, false},
		{"missing joined at", 200, `{"user":{"id":"987654321098765432"}}`, true, true, false},
		{"invalid joined at", 200, `{"user":{"id":"987654321098765432"},"joined_at":"invalid"}`, true, true, false},
		{"future joined at", 200, `{"user":{"id":"987654321098765432"},"joined_at":"2999-01-01T00:00:00Z"}`, true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: discordTestTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.Header.Get("Authorization") != "Bearer test-access-token" {
					t.Fatal("missing user access token")
				}
				switch r.URL.Path {
				case "/api/users/@me":
					return discordTestResponse(200, `{"id":"987654321098765432","username":"test","email":"test@example.com","verified":true}`), nil
				case "/api/v10/users/@me/guilds/" + guildID + "/member":
					if _, ok := r.Context().Deadline(); !ok {
						t.Fatal("member request needs a deadline")
					}
					return discordTestResponse(tc.status, tc.body), nil
				default:
					t.Fatalf("unexpected request: %s", r.URL.Path)
					return nil, nil
				}
			})}
			pc := &domain.ProviderConfig{Provider: domain.ProviderDiscord, Scopes: []string{"identify", "guilds.members.read"}, ExtraConfig: map[string]any{"guild_ids": []any{guildID, guildID}}}
			provider := buildProvider(pc, "client", "secret").(*OAuth2Provider)
			info, err := provider.ValidateToken(context.WithValue(context.Background(), oauth2.HTTPClient, client), "test-access-token")
			if err != nil {
				t.Fatal(err)
			}
			if requests != 2 {
				t.Fatalf("expected only user + one target guild request, got %d", requests)
			}
			if info.Email != "test@example.com" || info.RawProfile["created_at"] == nil {
				t.Fatal("lost existing user profile")
			}
			entry := info.RawProfile["guilds"].(map[string]any)[guildID].(map[string]any)
			member, known := entry["is_member"].(bool)
			if known != tc.known || member != tc.member {
				t.Fatalf("unexpected membership: %#v", entry)
			}
			_, joined := entry["joined_at"]
			if joined != tc.joined {
				t.Fatalf("unexpected joined_at: %#v", entry)
			}
			if _, err := time.Parse(time.RFC3339Nano, entry["checked_at"].(string)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDiscordSkipsUnrequestedGuildData(t *testing.T) {
	cases := []struct {
		name           string
		scopes, guilds []string
		token          *oauth2.Token
	}{
		{"no guild targets", []string{"identify", "guilds.members.read"}, nil, nil},
		{"no member scope", []string{"identify", "guilds"}, []string{"123456789012345678"}, nil},
		{"token lacks configured scope", []string{"identify", "guilds.members.read"}, []string{"123456789012345678"}, (&oauth2.Token{}).WithExtra(map[string]any{"scope": "identify guilds"})},
		{"invalid target", []string{"identify", "guilds.members.read"}, []string{"../other"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: discordTestTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.URL.Path != "/api/users/@me" {
					t.Fatalf("unexpected guild request: %s", r.URL)
				}
				return discordTestResponse(200, `{"id":"987654321098765432","username":"test"}`), nil
			})}
			provider := NewDiscordProviderWithGuilds("client", "secret", tc.scopes, tc.guilds)
			info, err := provider.fetchUser(context.Background(), client, tc.token)
			if err != nil {
				t.Fatal(err)
			}
			if requests != 1 || len(info.RawProfile["guilds"].(map[string]any)) != 0 {
				t.Fatal("unexpected guild snapshot")
			}
		})
	}
}

func TestDiscordMemberRequestCancellation(t *testing.T) {
	client := &http.Client{Transport: discordTestTransport(func(r *http.Request) (*http.Response, error) {
		<-r.Context().Done()
		return nil, r.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fetchDiscordGuildMember(ctx, client, "123456789012345678", "987654321098765432"); err == nil {
		t.Fatal("expected cancellation")
	}
}
