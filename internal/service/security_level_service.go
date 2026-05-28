package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	CurrentLevel int                           `json:"level"`
	MaxLevel     int                           `json:"max_level"`
	Bindings     []*domain.SocialBinding       `json:"bindings"`
	NextLevel    *NextLevelRequirement         `json:"next_level,omitempty"`
	History      []*domain.SecurityLevelChange `json:"history"`
}

type NextLevelRequirement struct {
	Level    int                `json:"level"`
	RuleName string             `json:"rule_name"`
	Missing  []MissingCondition `json:"missing"`
}

type MissingCondition struct {
	Type           domain.RuleConditionType `json:"type,omitempty"`
	Provider       string                   `json:"provider,omitempty"`
	Field          string                   `json:"field,omitempty"`
	Operator       string                   `json:"operator,omitempty"`
	Value          any                      `json:"value,omitempty"`
	Values         []string                 `json:"values,omitempty"`
	MinBindingDays int                      `json:"min_binding_days,omitempty"`
	IsBound        bool                     `json:"is_bound"`
	BoundDays      int                      `json:"bound_days"`
	IsSatisfied    bool                     `json:"is_satisfied"`
}

type ruleEvalContext struct {
	user        *domain.User
	bindings    map[string]*domain.SocialBinding
	bindingList []*domain.SocialBinding
	now         time.Time
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
	evalCtx := newRuleEvalContext(user, bindings, time.Now().UTC())

	newLevel := 0
	var matchedRuleID *uuid.UUID
	matchedName := ""
	for _, r := range rules {
		if s.evaluateRule(r.Conditions, evalCtx) {
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
			CreatedAt:     evalCtx.now,
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
			CreatedAt: evalCtx.now,
		})
	}

	return newLevel, nil
}

func (s *SecurityLevelService) evaluateRule(conds domain.RuleConditions, ctx ruleEvalContext) bool {
	// Support new nested structure (Items) if present
	if len(conds.Items) > 0 {
		return s.evaluateConditionItems(conds.Items, conds.Operator, ctx)
	}

	// Fallback to old flat structure (Conditions) for backward compatibility
	if len(conds.Conditions) == 0 {
		return false
	}
	op := conds.Operator
	if op == "" {
		op = domain.OperatorAND
	}
	matchedAny := false
	for _, c := range conds.Conditions {
		ok := evaluateCondition(c, ctx)
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

func (s *SecurityLevelService) evaluateConditionItems(items []domain.ConditionItem, operator domain.RuleOperator, ctx ruleEvalContext) bool {
	if len(items) == 0 {
		return false
	}

	op := operator
	if op == "" {
		op = domain.OperatorAND
	}

	matchedAny := false
	for _, item := range items {
		var ok bool

		// Evaluate nested group
		if item.Group != nil {
			ok = s.evaluateConditionItems(item.Group.Items, item.Group.Operator, ctx)
		} else if item.Condition != nil {
			// Evaluate single condition
			ok = evaluateCondition(*item.Condition, ctx)
		} else {
			// Empty item, skip
			continue
		}

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

func evaluateCondition(c domain.RuleCondition, ctx ruleEvalContext) bool {
	c = normalizeCondition(c)

	switch c.Type {
	case domain.ConditionProviderBound:
		return len(bindingCandidates(c, ctx)) > 0
	case domain.ConditionBindingAgeDays:
		for _, binding := range bindingCandidates(c, ctx) {
			boundDays := float64(daysSince(binding.BoundAt, ctx.now))
			if compareNumber(boundDays, float64(conditionDays(c)), c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionProviderAccountAgeDays:
		field := c.Field
		if field == "" {
			field = "created_at"
		}
		for _, binding := range bindingCandidates(c, ctx) {
			value, ok := rawValueAtPath(binding.RawProfile, field)
			if !ok {
				continue
			}
			createdAt, ok := parseRuleTime(value)
			accountDays := float64(daysSince(createdAt, ctx.now))
			if ok && compareNumber(accountDays, float64(conditionDays(c)), c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionProviderEmailVerified:
		for _, binding := range bindingCandidates(c, ctx) {
			actual, ok := providerEmailVerified(binding)
			if !ok {
				continue
			}
			expected := true
			if c.Value != nil {
				v, ok := toBool(c.Value)
				if !ok {
					return false
				}
				expected = v
			}
			if compareBool(actual, expected, c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionProviderEmailDomain:
		for _, binding := range bindingCandidates(c, ctx) {
			email := providerEmail(binding, c.Field)
			if emailDomainMatches(email, conditionStrings(c)) {
				return true
			}
		}
		return false
	case domain.ConditionProviderRawNumber:
		if c.Field == "" {
			return false
		}
		expected, ok := conditionNumber(c)
		if !ok {
			return false
		}
		for _, binding := range bindingCandidates(c, ctx) {
			actual, ok := rawNumber(binding.RawProfile, c.Field)
			if ok && compareNumber(actual, expected, c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionProviderRawString:
		if c.Field == "" {
			return false
		}
		for _, binding := range bindingCandidates(c, ctx) {
			actual, ok := rawString(binding.RawProfile, c.Field)
			if ok && compareString(actual, conditionStrings(c), c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionProviderRawBool:
		if c.Field == "" {
			return false
		}
		expected := true
		if c.Value != nil {
			v, ok := toBool(c.Value)
			if !ok {
				return false
			}
			expected = v
		}
		for _, binding := range bindingCandidates(c, ctx) {
			actual, ok := rawBool(binding.RawProfile, c.Field)
			if ok && compareBool(actual, expected, c.Operator) {
				return true
			}
		}
		return false
	case domain.ConditionUserEmailDomain:
		if ctx.user == nil {
			return false
		}
		return emailDomainMatches(ctx.user.Email, conditionStrings(c))
	case domain.ConditionUserCreatedAgeDays:
		if ctx.user == nil {
			return false
		}
		createdDays := float64(daysSince(ctx.user.CreatedAt, ctx.now))
		return compareNumber(createdDays, float64(conditionDays(c)), c.Operator)
	case domain.ConditionUserHasVerifiedEmail:
		if ctx.user == nil {
			return false
		}
		expected := true
		if c.Value != nil {
			v, ok := toBool(c.Value)
			if !ok {
				return false
			}
			expected = v
		}
		return compareBool(ctx.user.EmailVerified, expected, c.Operator)
	default:
		return false
	}
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

	evalCtx := newRuleEvalContext(user, bindings, time.Now().UTC())

	var next *NextLevelRequirement
	for _, r := range rules {
		if r.Level <= user.SecurityLevel {
			break
		}
		if !s.evaluateRule(r.Conditions, evalCtx) {
			var missing []MissingCondition
			// Support new nested structure
			if len(r.Conditions.Items) > 0 {
				missing = flattenConditionItems(r.Conditions.Items, evalCtx)
			} else {
				// Fallback to old flat structure
				missing = make([]MissingCondition, 0, len(r.Conditions.Conditions))
				for _, c := range r.Conditions.Conditions {
					missing = append(missing, conditionStatus(c, evalCtx))
				}
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
	// Support both new nested structure and old flat structure
	if len(r.Conditions.Items) == 0 && len(r.Conditions.Conditions) == 0 {
		return fmt.Errorf("%w: at least one condition required", ErrInvalidInput)
	}
	if r.Conditions.Operator == "" {
		r.Conditions.Operator = domain.OperatorAND
	}
	if r.Conditions.Operator != domain.OperatorAND && r.Conditions.Operator != domain.OperatorOR {
		return fmt.Errorf("%w: operator must be AND or OR", ErrInvalidInput)
	}

	// Validate new nested structure if present
	if len(r.Conditions.Items) > 0 {
		if err := validateConditionItems(r.Conditions.Items); err != nil {
			return err
		}
	}

	// Validate old flat structure if present
	for i, c := range r.Conditions.Conditions {
		c = normalizeCondition(c)
		if err := validateCondition(c); err != nil {
			return err
		}
		r.Conditions.Conditions[i] = c
	}
	return nil
}

func validateConditionItems(items []domain.ConditionItem) error {
	for _, item := range items {
		if item.Group != nil {
			// Validate nested group
			if item.Group.Operator != domain.OperatorAND && item.Group.Operator != domain.OperatorOR {
				return fmt.Errorf("%w: group operator must be AND or OR", ErrInvalidInput)
			}
			if len(item.Group.Items) == 0 {
				return fmt.Errorf("%w: group must have at least one item", ErrInvalidInput)
			}
			// Recursively validate nested items
			if err := validateConditionItems(item.Group.Items); err != nil {
				return err
			}
		} else if item.Condition != nil {
			// Validate single condition
			c := normalizeCondition(*item.Condition)
			if err := validateCondition(c); err != nil {
				return err
			}
			*item.Condition = c
		}
	}
	return nil
}

func validateCondition(c domain.RuleCondition) error {
	if c.MinBindingDays < 0 || c.MinDays < 0 {
		return fmt.Errorf("%w: condition days must be >= 0", ErrInvalidInput)
	}
	if providerCondition(c.Type) {
		if c.Provider != "" && !domain.IsValidProvider(c.Provider) {
			return fmt.Errorf("%w: unknown provider %q", ErrInvalidInput, c.Provider)
		}
	}
	switch c.Type {
	case domain.ConditionProviderBound,
		domain.ConditionBindingAgeDays,
		domain.ConditionProviderAccountAgeDays,
		domain.ConditionProviderEmailVerified,
		domain.ConditionProviderEmailDomain,
		domain.ConditionUserEmailDomain,
		domain.ConditionUserCreatedAgeDays,
		domain.ConditionUserHasVerifiedEmail:
		return nil
	case domain.ConditionProviderRawNumber,
		domain.ConditionProviderRawString,
		domain.ConditionProviderRawBool:
		if strings.TrimSpace(c.Field) == "" {
			return fmt.Errorf("%w: condition field required", ErrInvalidInput)
		}
		return nil
	default:
		return fmt.Errorf("%w: unsupported condition type %q", ErrInvalidInput, c.Type)
	}
}

func newRuleEvalContext(user *domain.User, bindings []*domain.SocialBinding, now time.Time) ruleEvalContext {
	activeBindings := activeSocialBindings(bindings)
	return ruleEvalContext{
		user:        user,
		bindings:    bindingMap(activeBindings),
		bindingList: activeBindings,
		now:         now,
	}
}

func activeSocialBindings(bindings []*domain.SocialBinding) []*domain.SocialBinding {
	out := make([]*domain.SocialBinding, 0, len(bindings))
	for _, b := range bindings {
		if b == nil {
			continue
		}
		if b.Status != "" && b.Status != domain.SocialBindingStatusActive {
			continue
		}
		out = append(out, b)
	}
	return out
}

func bindingMap(bindings []*domain.SocialBinding) map[string]*domain.SocialBinding {
	m := make(map[string]*domain.SocialBinding, len(bindings))
	for _, b := range bindings {
		if b == nil {
			continue
		}
		m[b.Provider] = b
	}
	return m
}

func bindingCandidates(c domain.RuleCondition, ctx ruleEvalContext) []*domain.SocialBinding {
	if c.Provider == "" {
		out := make([]*domain.SocialBinding, 0, len(ctx.bindingList))
		for _, b := range ctx.bindingList {
			if b != nil {
				out = append(out, b)
			}
		}
		return out
	}
	binding, ok := ctx.bindings[c.Provider]
	if !ok || binding == nil {
		return nil
	}
	return []*domain.SocialBinding{binding}
}

func normalizeCondition(c domain.RuleCondition) domain.RuleCondition {
	c.Provider = strings.TrimSpace(c.Provider)
	c.Field = strings.TrimSpace(c.Field)
	c.Operator = strings.ToLower(strings.TrimSpace(c.Operator))

	switch c.Type {
	case "":
		if c.MinBindingDays > 0 {
			c.Type = domain.ConditionBindingAgeDays
		} else {
			c.Type = domain.ConditionProviderBound
		}
	case "numeric_min":
		c.Type = domain.ConditionProviderRawNumber
		if c.Operator == "" {
			c.Operator = "gte"
		}
	case "boolean_equals":
		c.Type = domain.ConditionProviderRawBool
		if c.Operator == "" {
			c.Operator = "eq"
		}
	case "string_equals":
		c.Type = domain.ConditionProviderRawString
		if c.Operator == "" {
			c.Operator = "eq"
		}
	case "email_domain":
		if c.Provider == "" {
			c.Type = domain.ConditionUserEmailDomain
		} else {
			c.Type = domain.ConditionProviderEmailDomain
		}
	}

	if c.MinDays == 0 && c.MinBindingDays > 0 {
		c.MinDays = c.MinBindingDays
	}
	if c.Type == domain.ConditionBindingAgeDays && c.MinBindingDays == 0 {
		c.MinBindingDays = c.MinDays
	}
	if c.Type == domain.ConditionProviderAccountAgeDays && c.Field == "" {
		c.Field = "created_at"
	}
	if c.Operator == "" {
		switch c.Type {
		case domain.ConditionProviderRawNumber:
			c.Operator = "gte"
		case domain.ConditionProviderRawString:
			c.Operator = "eq"
		case domain.ConditionProviderRawBool,
			domain.ConditionProviderEmailVerified,
			domain.ConditionUserHasVerifiedEmail:
			c.Operator = "eq"
		}
	}
	return c
}

func providerCondition(t domain.RuleConditionType) bool {
	switch t {
	case domain.ConditionProviderBound,
		domain.ConditionBindingAgeDays,
		domain.ConditionProviderAccountAgeDays,
		domain.ConditionProviderEmailVerified,
		domain.ConditionProviderEmailDomain,
		domain.ConditionProviderRawNumber,
		domain.ConditionProviderRawString,
		domain.ConditionProviderRawBool:
		return true
	default:
		return false
	}
}

// flattenConditionItems recursively flattens nested condition items into a flat list
func flattenConditionItems(items []domain.ConditionItem, ctx ruleEvalContext) []MissingCondition {
	var result []MissingCondition
	for _, item := range items {
		if item.Group != nil {
			// Recursively flatten nested group
			result = append(result, flattenConditionItems(item.Group.Items, ctx)...)
		} else if item.Condition != nil {
			// Add single condition status
			result = append(result, conditionStatus(*item.Condition, ctx))
		}
	}
	return result
}

func conditionStatus(c domain.RuleCondition, ctx ruleEvalContext) MissingCondition {
	c = normalizeCondition(c)
	candidates := bindingCandidates(c, ctx)
	bound := len(candidates) > 0
	days := 0
	for _, binding := range candidates {
		if d := daysSince(binding.BoundAt, ctx.now); d > days {
			days = d
		}
	}
	minDays := conditionDays(c)
	return MissingCondition{
		Type:           c.Type,
		Provider:       c.Provider,
		Field:          c.Field,
		Operator:       c.Operator,
		Value:          c.Value,
		Values:         append([]string(nil), c.Values...),
		MinBindingDays: minDays,
		IsBound:        bound,
		BoundDays:      days,
		IsSatisfied:    evaluateCondition(c, ctx),
	}
}

func conditionDays(c domain.RuleCondition) int {
	if c.MinDays > 0 {
		return c.MinDays
	}
	if c.MinBindingDays > 0 {
		return c.MinBindingDays
	}
	if n, ok := conditionNumber(c); ok && n > 0 {
		return int(n)
	}
	return 0
}

func conditionNumber(c domain.RuleCondition) (float64, bool) {
	if c.Value != nil {
		return toFloat64(c.Value)
	}
	if c.MinDays > 0 {
		return float64(c.MinDays), true
	}
	if c.MinBindingDays > 0 {
		return float64(c.MinBindingDays), true
	}
	return 0, false
}

func conditionStrings(c domain.RuleCondition) []string {
	items := cleanStrings(c.Values)
	if len(items) > 0 {
		return items
	}
	return stringsFromValue(c.Value)
}

func providerEmailVerified(binding *domain.SocialBinding) (bool, bool) {
	if binding == nil {
		return false, false
	}
	for _, path := range []string{"email_verified", "primary_email_verified", "verified_email"} {
		if v, ok := rawBool(binding.RawProfile, path); ok {
			return v, true
		}
	}
	return false, false
}

func providerEmail(binding *domain.SocialBinding, field string) string {
	if binding == nil {
		return ""
	}
	if field != "" {
		if v, ok := rawString(binding.RawProfile, field); ok {
			return v
		}
	}
	if binding.ProviderEmail != nil {
		return *binding.ProviderEmail
	}
	for _, path := range []string{"email", "mail", "userPrincipalName"} {
		if v, ok := rawString(binding.RawProfile, path); ok {
			return v
		}
	}
	return ""
}

func emailDomainMatches(email string, domains []string) bool {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(email)), "@")
	if len(parts) != 2 || parts[1] == "" {
		return false
	}
	domainName := parts[1]
	for _, allowed := range domains {
		allowed = strings.TrimPrefix(strings.TrimSpace(strings.ToLower(allowed)), "@")
		if allowed != "" && domainName == allowed {
			return true
		}
	}
	return false
}

func rawNumber(data map[string]any, path string) (float64, bool) {
	v, ok := rawValueAtPath(data, path)
	if !ok {
		return 0, false
	}
	return toFloat64(v)
}

func rawString(data map[string]any, path string) (string, bool) {
	v, ok := rawValueAtPath(data, path)
	if !ok {
		return "", false
	}
	return toString(v)
}

func rawBool(data map[string]any, path string) (bool, bool) {
	v, ok := rawValueAtPath(data, path)
	if !ok {
		return false, false
	}
	return toBool(v)
}

func rawValueAtPath(data map[string]any, path string) (any, bool) {
	if data == nil || strings.TrimSpace(path) == "" {
		return nil, false
	}
	var current any = data
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, false
		}
		node, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := node[part]
		if !ok {
			return nil, false
		}
		current = v
	}
	return current, true
}

func parseRuleTime(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v.UTC(), !v.IsZero()
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return time.Time{}, false
		}
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t.UTC(), true
			}
		}
		return time.Time{}, false
	default:
		if n, ok := toFloat64(v); ok && n > 0 {
			return time.Unix(int64(n), 0).UTC(), true
		}
		return time.Time{}, false
	}
}

func daysSince(t time.Time, now time.Time) int {
	if t.IsZero() || now.Before(t) {
		return 0
	}
	return int(now.Sub(t.UTC()).Hours() / 24)
}

func compareNumber(actual, expected float64, op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "", "gte", ">=", "min":
		return actual >= expected
	case "gt", ">":
		return actual > expected
	case "lte", "<=":
		return actual <= expected
	case "lt", "<":
		return actual < expected
	case "eq", "=", "==":
		return math.Abs(actual-expected) < 0.0000001
	case "neq", "!=":
		return math.Abs(actual-expected) >= 0.0000001
	default:
		return false
	}
}

func compareString(actual string, expected []string, op string) bool {
	if len(expected) == 0 {
		return false
	}
	actual = strings.TrimSpace(actual)
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "", "eq", "=", "==":
		return actual == expected[0]
	case "neq", "!=":
		return actual != expected[0]
	case "contains":
		return strings.Contains(actual, expected[0])
	case "prefix", "starts_with":
		return strings.HasPrefix(actual, expected[0])
	case "suffix", "ends_with":
		return strings.HasSuffix(actual, expected[0])
	case "regex":
		matched, err := regexp.MatchString(expected[0], actual)
		return err == nil && matched
	case "in":
		for _, item := range expected {
			if actual == item {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func compareBool(actual, expected bool, op string) bool {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "", "eq", "=", "==":
		return actual == expected
	case "neq", "!=":
		return actual != expected
	default:
		return false
	}
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		n, err := v.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func toString(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v), true
	case fmt.Stringer:
		return strings.TrimSpace(v.String()), true
	case json.Number:
		return v.String(), true
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), true
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 64), true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		if n, ok := toFloat64(v); ok {
			return strconv.FormatFloat(n, 'f', -1, 64), true
		}
		return "", false
	case bool:
		return strconv.FormatBool(v), true
	default:
		return "", false
	}
}

func toBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "y", "on":
			return true, true
		case "false", "0", "no", "n", "off":
			return false, true
		default:
			return false, false
		}
	case float64:
		return v != 0, true
	case json.Number:
		n, err := v.Float64()
		return n != 0, err == nil
	default:
		return false, false
	}
}

func stringsFromValue(value any) []string {
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if strings.Contains(v, ",") {
			return cleanStrings(strings.Split(v, ","))
		}
		return cleanStrings([]string{v})
	case []string:
		return cleanStrings(v)
	case []any:
		items := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := toString(item); ok {
				items = append(items, s)
			}
		}
		return cleanStrings(items)
	default:
		if s, ok := toString(v); ok {
			return cleanStrings([]string{s})
		}
		return nil
	}
}

func cleanStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func sortRulesDesc(rules []*domain.SecurityLevelRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Level != rules[j].Level {
			return rules[i].Level > rules[j].Level
		}
		return rules[i].Priority > rules[j].Priority
	})
}
