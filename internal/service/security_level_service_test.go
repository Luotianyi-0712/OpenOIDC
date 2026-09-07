package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/google/uuid"
)

func TestDiscordGuildRuleConditions(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	const guildID = "123456789012345678"
	cases := []struct {
		name                              string
		profile                           map[string]any
		status                            domain.SocialBindingStatus
		member, nonmember, age, strictAge bool
	}{
		{"over sixty days", map[string]any{"is_member": true, "joined_at": now.Add(-60*24*time.Hour - time.Second).Format(time.RFC3339)}, domain.SocialBindingStatusActive, true, false, true, true},
		{"exactly sixty days", map[string]any{"is_member": true, "joined_at": now.Add(-60 * 24 * time.Hour).Format(time.RFC3339)}, domain.SocialBindingStatusActive, true, false, true, false},
		{"less than sixty days", map[string]any{"is_member": true, "joined_at": now.Add(-60*24*time.Hour + time.Second).Format(time.RFC3339)}, domain.SocialBindingStatusActive, true, false, false, false},
		{"left guild", map[string]any{"is_member": false, "joined_at": "2020-01-01T00:00:00Z"}, domain.SocialBindingStatusActive, false, true, false, false},
		{"unknown membership", map[string]any{"status": "unknown"}, domain.SocialBindingStatusActive, false, false, false, false},
		{"missing joined at", map[string]any{"is_member": true}, domain.SocialBindingStatusActive, true, false, false, false},
		{"invalid joined at", map[string]any{"is_member": true, "joined_at": "invalid"}, domain.SocialBindingStatusActive, true, false, false, false},
		{"future joined at", map[string]any{"is_member": true, "joined_at": now.Add(time.Hour).Format(time.RFC3339)}, domain.SocialBindingStatusActive, true, false, false, false},
		{"inactive binding", map[string]any{"is_member": true, "joined_at": "2020-01-01T00:00:00Z"}, domain.SocialBindingStatusUserUnbound, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Round-trip the snapshot as the repositories do when persisting JSON.
			encoded, _ := json.Marshal(map[string]any{"guilds": map[string]any{guildID: tc.profile}})
			var raw map[string]any
			if err := json.Unmarshal(encoded, &raw); err != nil {
				t.Fatal(err)
			}
			ctx := newRuleEvalContext(nil, []*domain.SocialBinding{{Provider: domain.ProviderDiscord, Status: tc.status, BoundAt: now.AddDate(-5, 0, 0), RawProfile: raw}}, now)
			conditions := []struct {
				rule domain.RuleCondition
				want bool
			}{
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: guildID}, tc.member},
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: guildID, Value: false}, tc.nonmember},
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: guildID, Operator: "neq", Value: false}, tc.member},
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: guildID, MinDays: 60}, tc.age},
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: guildID, MinDays: 60, Operator: "gt"}, tc.strictAge},
				{domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "111111111111111111"}, false},
			}
			for _, c := range conditions {
				if got := evaluateCondition(c.rule, ctx); got != c.want {
					t.Errorf("%+v: got %t, want %t", c.rule, got, c.want)
				}
			}
			ageRule := conditions[3].rule
			if !tc.age {
				ageRule.Operator = "lt"
				if (tc.name == "future joined at" || tc.name == "invalid joined at" || tc.name == "missing joined at") && evaluateCondition(ageRule, ctx) {
					t.Fatal("invalid timestamp matched a less-than rule")
				}
			}
		})
	}
}

func TestValidateDiscordGuildRules(t *testing.T) {
	for _, tc := range []struct {
		name  string
		c     domain.RuleCondition
		valid bool
	}{
		{"member", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "123456789012345678"}, true},
		{"age", domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: "123456789012345678", MinDays: 60}, true},
		{"missing id", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember}, false},
		{"invalid id", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "a.b"}, false},
		{"overflow id", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "18446744073709551616"}, false},
		{"wrong provider", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Provider: "github", Field: "123456789012345678"}, false},
		{"invalid bool", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "123456789012345678", Value: "unknown"}, false},
		{"invalid operator", domain.RuleCondition{Type: domain.ConditionDiscordGuildMember, Field: "123456789012345678", Operator: "gt"}, false},
		{"negative days", domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: "123456789012345678", Value: -1}, false},
		{"fractional days", domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: "123456789012345678", Value: 1.5}, false},
		{"overflow days", domain.RuleCondition{Type: domain.ConditionDiscordGuildAgeDays, Field: "123456789012345678", Value: 1e100}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule := &domain.SecurityLevelRule{Name: "Discord rule", Conditions: domain.RuleConditions{Items: []domain.ConditionItem{{Group: &domain.ConditionGroup{Operator: domain.OperatorAND, Items: []domain.ConditionItem{{Condition: &tc.c}}}}}}}
			if err := validateRule(rule); (err == nil) != tc.valid {
				t.Fatalf("valid=%t, err=%v", tc.valid, err)
			}
		})
	}
}

func TestEvaluateConditionLegacyBindingAgeRule(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -45),
		},
	}, now)

	condition := domain.RuleCondition{
		Provider:       domain.ProviderGitHub,
		MinBindingDays: 30,
	}

	if !evaluateCondition(condition, ctx) {
		t.Fatal("expected legacy provider + min_binding_days rule to match active binding age")
	}
}

func TestEvaluateConditionUsesProviderAccountCreatedAt(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -1),
			RawProfile: map[string]any{
				"created_at": "2025-01-01T00:00:00Z",
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderAccountAgeDays,
		Provider: domain.ProviderGitHub,
		MinDays:  300,
	}

	if !evaluateCondition(condition, ctx) {
		t.Fatal("expected provider_account_age_days to use RawProfile created_at, not binding time")
	}
}

func TestEvaluateConditionProviderAccountCreatedAtMissingFails(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -400),
			RawProfile: map[string]any{
				"followers": 10,
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderAccountAgeDays,
		Provider: domain.ProviderGitHub,
		MinDays:  300,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected missing provider created_at to fail instead of falling back to binding time")
	}
}

func TestEvaluateConditionProviderEmailVerifiedUnknownDoesNotMatchFalse(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -30),
			RawProfile: map[string]any{
				"email": "user@example.com",
			},
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderEmailVerified,
		Provider: domain.ProviderGitHub,
		Operator: "eq",
		Value:    false,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected unknown email verification state to be non-matching, not false")
	}
}

func TestEvaluateConditionIgnoresInactiveBindings(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(nil, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusUserUnbound,
			BoundAt:  now.AddDate(0, 0, -400),
		},
	}, now)

	condition := domain.RuleCondition{
		Type:     domain.ConditionProviderBound,
		Provider: domain.ProviderGitHub,
	}

	if evaluateCondition(condition, ctx) {
		t.Fatal("expected inactive social binding to be ignored")
	}
}

func TestEvaluateConditionUsesAgeComparisonOperator(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx := newRuleEvalContext(&domain.User{CreatedAt: now.AddDate(0, 0, -5)}, []*domain.SocialBinding{
		{
			ID:       uuid.New(),
			Provider: domain.ProviderGitHub,
			Status:   domain.SocialBindingStatusActive,
			BoundAt:  now.AddDate(0, 0, -5),
			RawProfile: map[string]any{
				"created_at": now.AddDate(0, 0, -5).Format(time.RFC3339),
			},
		},
	}, now)

	cases := []struct {
		name      string
		condition domain.RuleCondition
	}{
		{
			name: "binding age lte",
			condition: domain.RuleCondition{
				Type:     domain.ConditionBindingAgeDays,
				Provider: domain.ProviderGitHub,
				Operator: "lte",
				MinDays:  7,
			},
		},
		{
			name: "provider account age lt",
			condition: domain.RuleCondition{
				Type:     domain.ConditionProviderAccountAgeDays,
				Provider: domain.ProviderGitHub,
				Operator: "lt",
				MinDays:  7,
			},
		},
		{
			name: "user age lte",
			condition: domain.RuleCondition{
				Type:     domain.ConditionUserCreatedAgeDays,
				Operator: "lte",
				MinDays:  7,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !evaluateCondition(tc.condition, ctx) {
				t.Fatalf("expected %s to honor comparison operator", tc.name)
			}
		})
	}
}
