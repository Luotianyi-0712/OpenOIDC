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

type RuleCondition struct {
	Provider       string `json:"provider"`
	MinBindingDays int    `json:"min_binding_days"`
}

type RuleConditions struct {
	Operator   RuleOperator    `json:"operator"`
	Conditions []RuleCondition `json:"rules"`
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
