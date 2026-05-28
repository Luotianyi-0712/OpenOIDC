package social

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

// discordEpochMs is Discord's snowflake epoch — first ms of 2015-01-01 UTC.
const discordEpochMs = 1420070400000

// snowflakeCreatedAt decodes the timestamp embedded in a Discord snowflake ID.
// Bits 22..63 hold milliseconds since the Discord epoch.
func snowflakeCreatedAt(id string) (time.Time, bool) {
	n, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	ms := int64(n>>22) + discordEpochMs
	return time.UnixMilli(ms).UTC(), true
}

type discordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	GlobalName    string `json:"global_name"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	MFAEnabled    bool   `json:"mfa_enabled"`
	PremiumType   int    `json:"premium_type"`
	PublicFlags   int    `json:"public_flags"`
	Flags         int    `json:"flags"`
	Locale        string `json:"locale"`
}

func NewDiscordProvider(clientID, clientSecret string, scopes []string) *OAuth2Provider {
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}
	return &OAuth2Provider{
		name: domain.ProviderDiscord,
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://discord.com/api/oauth2/authorize",
				TokenURL: "https://discord.com/api/oauth2/token",
			},
			Scopes: scopes,
		},
		userURL: "https://discord.com/api/users/@me",
		parseUser: func(body []byte) (*port.ProviderUserInfo, error) {
			var u discordUser
			if err := json.Unmarshal(body, &u); err != nil {
				return nil, fmt.Errorf("decode discord user: %w", err)
			}
			display := u.GlobalName
			if display == "" {
				display = u.Username
			}
			avatar := ""
			if u.Avatar != "" {
				ext := "png"
				if len(u.Avatar) >= 2 && u.Avatar[:2] == "a_" {
					ext = "gif"
				}
				avatar = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.%s", u.ID, u.Avatar, ext)
			}
			var raw map[string]any
			_ = json.Unmarshal(body, &raw)
			if raw == nil {
				raw = map[string]any{}
			}
			raw["email_verified"] = u.Verified
			if created, ok := snowflakeCreatedAt(u.ID); ok {
				raw["created_at"] = created.Format(time.RFC3339)
			}
			raw = normalizeRawProfile(raw, u.Email)
			return &port.ProviderUserInfo{
				ProviderUID:   u.ID,
				Email:         u.Email,
				EmailVerified: u.Verified,
				DisplayName:   display,
				AvatarURL:     avatar,
				RawProfile:    raw,
			}, nil
		},
	}
}
