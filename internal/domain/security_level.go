package domain

import (
	"time"

	"github.com/google/uuid"
)

type RuleOperator string

const (
	OperatorAND RuleOperator = "AND"
	OperatorOR  RuleOperator = "OR"
)

type RuleConditionType string

const (
	ConditionProviderBound          RuleConditionType = "provider_bound"
	ConditionBindingAgeDays         RuleConditionType = "binding_age_days"
	ConditionProviderAccountAgeDays RuleConditionType = "provider_account_age_days"
	ConditionProviderEmailVerified  RuleConditionType = "provider_email_verified"
	ConditionProviderEmailDomain    RuleConditionType = "provider_email_domain"
	ConditionProviderRawNumber      RuleConditionType = "provider_raw_number"
	ConditionProviderRawString      RuleConditionType = "provider_raw_string"
	ConditionProviderRawBool        RuleConditionType = "provider_raw_bool"
	ConditionUserEmailDomain        RuleConditionType = "user_email_domain"
	ConditionUserCreatedAgeDays     RuleConditionType = "user_created_age_days"
	ConditionUserHasVerifiedEmail   RuleConditionType = "user_has_verified_email"
)

type RuleCondition struct {
	Type           RuleConditionType `json:"type,omitempty"`
	Provider       string            `json:"provider,omitempty"`
	Field          string            `json:"field,omitempty"`
	Operator       string            `json:"operator,omitempty"`
	Value          any               `json:"value,omitempty"`
	Values         []string          `json:"values,omitempty"`
	MinDays        int               `json:"min_days,omitempty"`
	MinBindingDays int               `json:"min_binding_days,omitempty"`
}

// ConditionItem can be either a RuleCondition or a nested ConditionGroup
type ConditionItem struct {
	Condition *RuleCondition   `json:"condition,omitempty"`
	Group     *ConditionGroup  `json:"group,omitempty"`
}

type ConditionGroup struct {
	Operator RuleOperator    `json:"operator"`
	Items    []ConditionItem `json:"items"`
}

type RuleConditions struct {
	Operator   RuleOperator    `json:"operator"`
	Conditions []RuleCondition `json:"rules"` // Keep for backward compatibility
	Items      []ConditionItem `json:"items,omitempty"` // New nested structure
}

type SecurityLevelRule struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Level       int            `json:"level"`
	Priority    int            `json:"priority"`
	Conditions  RuleConditions `json:"conditions"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type SecurityLevelChange struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	OldLevel      int        `json:"old_level"`
	NewLevel      int        `json:"new_level"`
	Reason        string     `json:"reason"`
	MatchedRuleID *uuid.UUID `json:"matched_rule_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}
