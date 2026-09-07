package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

type discordGuildMember struct {
	JoinedAt string `json:"joined_at"`
	User     *struct {
		ID string `json:"id"`
	} `json:"user"`
}

func NewDiscordProvider(clientID, clientSecret string, scopes []string) *OAuth2Provider {
	return NewDiscordProviderWithGuilds(clientID, clientSecret, scopes, nil)
}

func NewDiscordProviderWithGuilds(clientID, clientSecret string, scopes, targetGuildIDs []string) *OAuth2Provider {
	// Default scopes if not configured
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}
	provider := &OAuth2Provider{
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
	provider.fetchUser = func(ctx context.Context, client *http.Client, token *oauth2.Token) (*port.ProviderUserInfo, error) {
		body, err := doGet(ctx, client, provider.userURL)
		if err != nil {
			return nil, fmt.Errorf("fetch discord user info: %w", err)
		}
		info, err := provider.parseUser(body)
		if err != nil {
			return nil, err
		}
		// Always replace the snapshot, including when permissions have been removed.
		info.RawProfile["guilds"] = map[string]any{}
		grantedScopes := scopes
		if token != nil {
			if granted, ok := token.Extra("scope").(string); ok {
				grantedScopes = strings.Fields(granted)
			}
		}
		if !hasDiscordScope(grantedScopes, "guilds.members.read") || len(targetGuildIDs) == 0 {
			return info, nil
		}
		guildCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		defer cancel()
		guilds := make(map[string]any, len(targetGuildIDs))
		for _, guildID := range targetGuildIDs {
			guildID = strings.TrimSpace(guildID)
			if !domain.IsDiscordGuildID(guildID) || guilds[guildID] != nil {
				continue
			}
			entry, memberErr := fetchDiscordGuildMember(guildCtx, client, guildID, info.ProviderUID)
			if memberErr != nil {
				slog.Warn("discord guild membership unavailable", "guild_id", guildID, "error", memberErr)
				entry = map[string]any{"status": "unknown"}
			}
			entry["checked_at"] = time.Now().UTC().Format(time.RFC3339Nano)
			guilds[guildID] = entry
		}
		info.RawProfile["guilds"] = guilds
		return info, nil
	}
	return provider
}

func hasDiscordScope(scopes []string, wanted string) bool {
	for _, scope := range scopes {
		if strings.EqualFold(strings.TrimSpace(scope), wanted) {
			return true
		}
	}
	return false
}

func fetchDiscordGuildMember(ctx context.Context, client *http.Client, guildID, userID string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://discord.com/api/v10/users/@me/guilds/"+guildID+"/member", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		var apiError struct {
			Code int `json:"code"`
		}
		if json.Unmarshal(body, &apiError) == nil && (apiError.Code == 10007 || apiError.Code == 10004) {
			return map[string]any{"is_member": false}, nil
		}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord member http %d", resp.StatusCode)
	}
	var member discordGuildMember
	if err := json.Unmarshal(body, &member); err != nil {
		return nil, fmt.Errorf("decode discord member: %w", err)
	}
	if member.User == nil || member.User.ID != userID {
		return nil, fmt.Errorf("discord member user mismatch")
	}
	entry := map[string]any{"is_member": true}
	if joined, err := time.Parse(time.RFC3339Nano, member.JoinedAt); err == nil && !joined.IsZero() && !joined.After(time.Now()) {
		entry["joined_at"] = joined.UTC().Format(time.RFC3339Nano)
	}
	return entry, nil
}
