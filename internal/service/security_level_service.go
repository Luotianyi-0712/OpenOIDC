package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type SecurityLevelService struct {
	ruleRepo    port.RuleRepository
	bindingRepo port.BindingRepository
	userRepo    port.UserRepository
	auditRepo   port.AuditRepository
}

func NewSecurityLevelService(
	ruleRepo port.RuleRepository,
	bindingRepo port.BindingRepository,
	userRepo port.UserRepository,
	auditRepo port.AuditRepository,
) *SecurityLevelService {
	return &SecurityLevelService{
		ruleRepo:    ruleRepo,
		bindingRepo: bindingRepo,
		userRepo:    userRepo,
		auditRepo:   auditRepo,
	}
}

type LevelInfo struct {
	CurrentLevel int                          `json:"level"`
	MaxLevel     int                          `json:"max_level"`
	Bindings     []*domain.SocialBinding      `json:"bindings"`
	NextLevel    *NextLevelRequirement        `json:"next_level,omitempty"`
	History      []*domain.SecurityLevelChange `json:"history"`
}

type NextLevelRequirement struct {
	Level    int                `json:"level"`
	RuleName string             `json:"rule_name"`
	Missing  []MissingCondition `json:"missing"`
}

type MissingCondition struct {
	Provider       string `json:"provider"`
	MinBindingDays int    `json:"min_binding_days"`
	IsBound        bool   `json:"is_bound"`
	BoundDays      int    `json:"bound_days"`
}

func (s *SecurityLevelService) ComputeSecurityLevel(ctx context.Context, userID uuid.UUID) (int, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("lookup user: %w", err)
	}

	rules, err := s.ruleRepo.ListActive(ctx)
	if err != nil {
		return 0, fmt.Errorf("list rules: %w", err)
	}
	sortRulesDesc(rules)

	bindings, err := s.bindingRepo.ListByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("list bindings: %w", err)
	}
	bindMap := bindingMap(bindings)

	now := time.Now().UTC()
	newLevel := 0
	var matchedRuleID *uuid.UUID
	matchedName := ""
	for _, r := range rules {
		if s.evaluateRule(r.Conditions, bindMap, now) {
			newLevel = r.Level
			rid := r.ID
			matchedRuleID = &rid
			matchedName = r.Name
			break
		}
	}

	if newLevel != user.SecurityLevel {
		oldLevel := user.SecurityLevel
		if err := s.userRepo.UpdateSecurityLevel(ctx, userID, newLevel); err != nil {
			return 0, fmt.Errorf("update security level: %w", err)
		}
		change := &domain.SecurityLevelChange{
			ID:            uuid.New(),
			UserID:        userID,
			OldLevel:      oldLevel,
			NewLevel:      newLevel,
			Reason:        fmt.Sprintf("rule_evaluation: %s", matchedName),
			MatchedRuleID: matchedRuleID,
			CreatedAt:     now,
		}
		if err := s.auditRepo.CreateSecurityLevelChange(ctx, change); err != nil {
			return 0, fmt.Errorf("record level change: %w", err)
		}
		action := "security_level.upgraded"
		if newLevel < oldLevel {
			action = "security_level.downgraded"
		}
		rt := "user"
		rid := userID.String()
		_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       action,
			ResourceType: &rt,
			ResourceID:   &rid,
			Details: map[string]any{
				"old_level": oldLevel,
				"new_level": newLevel,
				"rule":      matchedName,
			},
			CreatedAt: now,
		})
	}

	return newLevel, nil
}

func (s *SecurityLevelService) evaluateRule(conds domain.RuleConditions, bindings map[string]time.Time, now time.Time) bool {
	if len(conds.Conditions) == 0 {
		return false
	}
	op := conds.Operator
	if op == "" {
		op = domain.OperatorAND
	}
	matchedAny := false
	for _, c := range conds.Conditions {
		ok := evaluateCondition(c, bindings, now)
		if op == domain.OperatorOR {
			if ok {
				return true
			}
		} else {
			if !ok {
				return false
			}
			matchedAny = true
		}
	}
	if op == domain.OperatorOR {
		return false
	}
	return matchedAny
}

func evaluateCondition(c domain.RuleCondition, bindings map[string]time.Time, now time.Time) bool {
	boundAt, ok := bindings[c.Provider]
	if !ok {
		return false
	}
	if c.MinBindingDays <= 0 {
		return true
	}
	days := int(now.Sub(boundAt).Hours() / 24)
	return days >= c.MinBindingDays
}

func (s *SecurityLevelService) RecomputeAll(ctx context.Context) error {
	const pageSize = 200
	offset := 0
	for {
		users, _, err := s.userRepo.List(ctx, port.ListUsersOptions{Offset: offset, Limit: pageSize})
		if err != nil {
			return fmt.Errorf("list users: %w", err)
		}
		if len(users) == 0 {
			return nil
		}
		for _, u := range users {
			if u.Status == domain.UserStatusDeleted {
				continue
			}
			if _, err := s.ComputeSecurityLevel(ctx, u.ID); err != nil {
				return fmt.Errorf("compute level for %s: %w", u.ID, err)
			}
		}
		if len(users) < pageSize {
			return nil
		}
		offset += pageSize
	}
}

func (s *SecurityLevelService) GetLevelInfo(ctx context.Context, userID uuid.UUID) (*LevelInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	bindings, err := s.bindingRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list bindings: %w", err)
	}
	rules, err := s.ruleRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list rules: %w", err)
	}
	sortRulesDesc(rules)

	maxLevel := 0
	for _, r := range rules {
		if r.Level > maxLevel {
			maxLevel = r.Level
		}
	}
	if maxLevel == 0 {
		maxLevel = 1
	}

	bindMap := bindingMap(bindings)
	now := time.Now().UTC()

	var next *NextLevelRequirement
	for _, r := range rules {
		if r.Level <= user.SecurityLevel {
			break
		}
		if !s.evaluateRule(r.Conditions, bindMap, now) {
			missing := make([]MissingCondition, 0, len(r.Conditions.Conditions))
			for _, c := range r.Conditions.Conditions {
				boundAt, ok := bindMap[c.Provider]
				days := 0
				if ok {
					days = int(now.Sub(boundAt).Hours() / 24)
				}
				missing = append(missing, MissingCondition{
					Provider:       c.Provider,
					MinBindingDays: c.MinBindingDays,
					IsBound:        ok,
					BoundDays:      days,
				})
			}
			next = &NextLevelRequirement{
				Level:    r.Level,
				RuleName: r.Name,
				Missing:  missing,
			}
		}
	}

	history, err := s.auditRepo.ListSecurityLevelChanges(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}

	return &LevelInfo{
		CurrentLevel: user.SecurityLevel,
		MaxLevel:     maxLevel,
		Bindings:     bindings,
		NextLevel:    next,
		History:      history,
	}, nil
}

func (s *SecurityLevelService) CreateRule(ctx context.Context, r *domain.SecurityLevelRule) error {
	if err := validateRule(r); err != nil {
		return err
	}
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	now := time.Now().UTC()
	r.CreatedAt = now
	r.UpdatedAt = now
	return s.ruleRepo.Create(ctx, r)
}

func (s *SecurityLevelService) UpdateRule(ctx context.Context, r *domain.SecurityLevelRule) error {
	if err := validateRule(r); err != nil {
		return err
	}
	r.UpdatedAt = time.Now().UTC()
	return s.ruleRepo.Update(ctx, r)
}

func (s *SecurityLevelService) DeleteRule(ctx context.Context, id uuid.UUID) error {
	return s.ruleRepo.Delete(ctx, id)
}

func (s *SecurityLevelService) ListRules(ctx context.Context) ([]*domain.SecurityLevelRule, error) {
	return s.ruleRepo.ListAll(ctx)
}

func (s *SecurityLevelService) GetRule(ctx context.Context, id uuid.UUID) (*domain.SecurityLevelRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rule, nil
}

func validateRule(r *domain.SecurityLevelRule) error {
	if r.Name == "" {
		return fmt.Errorf("%w: rule name required", ErrInvalidInput)
	}
	if r.Level < 0 {
		return fmt.Errorf("%w: level must be >= 0", ErrInvalidInput)
	}
	if len(r.Conditions.Conditions) == 0 {
		return fmt.Errorf("%w: at least one condition required", ErrInvalidInput)
	}
	if r.Conditions.Operator != domain.OperatorAND && r.Conditions.Operator != domain.OperatorOR {
		return fmt.Errorf("%w: operator must be AND or OR", ErrInvalidInput)
	}
	for _, c := range r.Conditions.Conditions {
		if c.Provider == "" {
			return fmt.Errorf("%w: condition provider required", ErrInvalidInput)
		}
		if !domain.IsValidProvider(c.Provider) {
			return fmt.Errorf("%w: unknown provider %q", ErrInvalidInput, c.Provider)
		}
		if c.MinBindingDays < 0 {
			return fmt.Errorf("%w: min_binding_days must be >= 0", ErrInvalidInput)
		}
	}
	return nil
}

func bindingMap(bindings []*domain.SocialBinding) map[string]time.Time {
	m := make(map[string]time.Time, len(bindings))
	for _, b := range bindings {
		m[b.Provider] = b.BoundAt
	}
	return m
}

func sortRulesDesc(rules []*domain.SecurityLevelRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Level != rules[j].Level {
			return rules[i].Level > rules[j].Level
		}
		return rules[i].Priority > rules[j].Priority
	})
}
