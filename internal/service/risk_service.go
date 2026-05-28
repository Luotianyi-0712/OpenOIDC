package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type RiskService struct {
	reportRepo   port.RiskReportRepository
	riskListRepo port.RiskListRepository
	bindingRepo  port.BindingRepository
	userRepo     port.UserRepository
	auditRepo    port.AuditRepository
	securitySvc  *SecurityLevelService
	emailSvc     port.EmailSender
}

func NewRiskService(
	reportRepo port.RiskReportRepository,
	riskListRepo port.RiskListRepository,
	bindingRepo port.BindingRepository,
	userRepo port.UserRepository,
	auditRepo port.AuditRepository,
	securitySvc *SecurityLevelService,
	emailSvc port.EmailSender,
) *RiskService {
	return &RiskService{
		reportRepo:   reportRepo,
		riskListRepo: riskListRepo,
		bindingRepo:  bindingRepo,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		securitySvc:  securitySvc,
		emailSvc:     emailSvc,
	}
}

// ReportUser allows a developer to report an abusive user of their app.
func (s *RiskService) ReportUser(ctx context.Context, clientID, reporterID, targetID uuid.UUID, reason, category string) (*domain.RiskReport, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: reason required", ErrInvalidInput)
	}
	switch category {
	case domain.ReportCategorySpam, domain.ReportCategoryAbuse, domain.ReportCategoryFraud, domain.ReportCategoryBot, domain.ReportCategoryOther:
	default:
		category = domain.ReportCategoryOther
	}

	existingReports, err := s.reportRepo.ListByTarget(ctx, targetID)
	if err != nil {
		return nil, fmt.Errorf("list target reports: %w", err)
	}
	for _, existing := range existingReports {
		if existing.ClientID == clientID && existing.ReporterID == reporterID && existing.Status != domain.ReportStatusDismissed {
			return nil, ErrAlreadyExists
		}
	}

	report := &domain.RiskReport{
		ID:         uuid.New(),
		ClientID:   clientID,
		ReporterID: reporterID,
		TargetID:   targetID,
		Reason:     reason,
		Category:   category,
		Status:     domain.ReportStatusPending,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	rt := "risk_report"
	rid := report.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &reporterID,
		Action:       "risk.user_reported",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"target": targetID.String(), "category": category},
		CreatedAt:    report.CreatedAt,
	})

	return report, nil
}

// ReportApp allows a user to report an abusive or malicious app.
func (s *RiskService) ReportApp(ctx context.Context, reporterID, targetClientID uuid.UUID, reason, category string) (*domain.RiskReport, error) {
	if reason == "" {
		return nil, fmt.Errorf("%w: reason required", ErrInvalidInput)
	}
	switch category {
	case domain.ReportCategorySpam, domain.ReportCategoryAbuse, domain.ReportCategoryFraud, domain.ReportCategoryBot, domain.ReportCategoryOther:
	default:
		category = domain.ReportCategoryOther
	}

	// Check if user has already reported this app
	existingReports, err := s.reportRepo.ListByTarget(ctx, targetClientID)
	if err != nil {
		return nil, fmt.Errorf("list target reports: %w", err)
	}
	for _, existing := range existingReports {
		if existing.ReporterID == reporterID && existing.Status != domain.ReportStatusDismissed {
			return nil, ErrAlreadyExists
		}
	}

	report := &domain.RiskReport{
		ID:         uuid.New(),
		ClientID:   uuid.Nil, // No client context for user-reported apps
		ReporterID: reporterID,
		TargetID:   targetClientID,
		Reason:     reason,
		Category:   category,
		Status:     domain.ReportStatusPending,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.reportRepo.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("create report: %w", err)
	}

	rt := "risk_report"
	rid := report.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &reporterID,
		Action:       "risk.app_reported",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"target": targetClientID.String(), "category": category},
		CreatedAt:    report.CreatedAt,
	})

	return report, nil
}

// ConfirmReport is called by admin to confirm a report and enforce risk measures.
func (s *RiskService) ConfirmReport(ctx context.Context, reportID, adminID uuid.UUID, note string) error {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("get report: %w", err)
	}
	if report.Status != domain.ReportStatusPending {
		return nil
	}

	now := time.Now().UTC()
	report.Status = domain.ReportStatusConfirmed
	report.AdminNote = note
	report.ResolvedAt = &now
	report.ResolvedBy = &adminID
	if err := s.reportRepo.Update(ctx, report); err != nil {
		return fmt.Errorf("update report: %w", err)
	}
	rt := "risk_report"
	rid := report.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &adminID,
		Action:       "risk.report_confirmed",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"target": report.TargetID.String(), "note": note},
		CreatedAt:    now,
	})

	// Notify the reporter
	s.notifyReporter(ctx, report, "confirmed", note)

	// Only enforce risk measures if this is a user report (ClientID != uuid.Nil)
	// For app reports (ClientID == uuid.Nil), admin should manually disable the app
	if report.ClientID != uuid.Nil {
		return s.enforceRisk(ctx, report.TargetID, report.ID)
	}
	return nil
}

// DismissReport marks a report as dismissed (false positive).
func (s *RiskService) DismissReport(ctx context.Context, reportID, adminID uuid.UUID, note string) error {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("get report: %w", err)
	}
	wasConfirmed := report.Status == domain.ReportStatusConfirmed
	now := time.Now().UTC()
	report.Status = domain.ReportStatusDismissed
	report.AdminNote = note
	report.ResolvedAt = &now
	report.ResolvedBy = &adminID
	if err := s.reportRepo.Update(ctx, report); err != nil {
		return err
	}
	rt := "risk_report"
	rid := report.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &adminID,
		Action:       "risk.report_dismissed",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"target": report.TargetID.String(), "note": note, "was_confirmed": wasConfirmed},
		CreatedAt:    now,
	})

	// Notify the reporter
	s.notifyReporter(ctx, report, "dismissed", note)

	if wasConfirmed {
		if err := s.riskListRepo.DeleteByReport(ctx, report.ID); err != nil {
			return fmt.Errorf("delete report risk entries: %w", err)
		}
		return s.restoreUserIfRiskCleared(ctx, report.TargetID)
	}
	return nil
}

// enforceRisk lowers the user's trust level to 0, suspends the account, and adds their social bindings to the risk list.
func (s *RiskService) enforceRisk(ctx context.Context, userID uuid.UUID, reportID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	oldLevel := user.SecurityLevel
	if err := s.userRepo.UpdateSecurityLevel(ctx, userID, 0); err != nil {
		return fmt.Errorf("downgrade user: %w", err)
	}

	if user.Status == domain.UserStatusActive {
		user.SecurityLevel = 0
		user.Status = domain.UserStatusSuspended
		_ = s.userRepo.Update(ctx, user)
	}

	_ = s.auditRepo.CreateSecurityLevelChange(ctx, &domain.SecurityLevelChange{
		ID:        uuid.New(),
		UserID:    userID,
		OldLevel:  oldLevel,
		NewLevel:  0,
		Reason:    "risk_enforcement",
		CreatedAt: time.Now().UTC(),
	})

	// Add all user's social bindings to the risk list.
	bindings, err := s.bindingRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list bindings: %w", err)
	}
	for _, b := range bindings {
		existing, _ := s.riskListRepo.Check(ctx, b.Provider, b.ProviderUID)
		if existing != nil {
			continue
		}
		_ = s.riskListRepo.Add(ctx, &domain.RiskListEntry{
			ID:          uuid.New(),
			Provider:    b.Provider,
			ProviderUID: b.ProviderUID,
			UserID:      &userID,
			Reason:      "risk_enforcement: confirmed abuse report",
			ReportID:    &reportID,
			CreatedAt:   time.Now().UTC(),
		})
	}

	rt := "user"
	rid := userID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &userID,
		Action:       "risk.enforcement",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"bindings_blacklisted": len(bindings)},
		CreatedAt:    time.Now().UTC(),
	})

	return nil
}

// CheckRisk checks if a social account is in the risk list.
func (s *RiskService) CheckRisk(ctx context.Context, provider, providerUID string) (*domain.RiskListEntry, error) {
	return s.riskListRepo.Check(ctx, provider, providerUID)
}

func (s *RiskService) AddToRiskList(ctx context.Context, provider, providerUID, reason string, userID, adminID *uuid.UUID) (*domain.RiskListEntry, error) {
	provider = strings.TrimSpace(provider)
	providerUID = strings.TrimSpace(providerUID)
	reason = strings.TrimSpace(reason)
	if provider == "" {
		return nil, fmt.Errorf("%w: provider required", ErrInvalidInput)
	}
	if providerUID == "" {
		return nil, fmt.Errorf("%w: provider_uid required", ErrInvalidInput)
	}
	if reason == "" {
		return nil, fmt.Errorf("%w: reason required", ErrInvalidInput)
	}
	if _, err := s.riskListRepo.Check(ctx, provider, providerUID); err == nil {
		return nil, ErrAlreadyExists
	} else if !errors.Is(err, port.ErrNotFound) {
		return nil, err
	}

	entry := &domain.RiskListEntry{
		ID:          uuid.New(),
		Provider:    provider,
		ProviderUID: providerUID,
		UserID:      userID,
		Reason:      reason,
		AddedBy:     adminID,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.riskListRepo.Add(ctx, entry); err != nil {
		return nil, fmt.Errorf("add risk entry: %w", err)
	}

	rt := "risk_list"
	rid := entry.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       adminID,
		Action:       "risk.list_added",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"provider": provider, "provider_uid": providerUID, "user_id": userID, "reason": reason},
		CreatedAt:    entry.CreatedAt,
	})
	return entry, nil
}

// ListRiskEntries returns the risk blacklist.
func (s *RiskService) ListRiskEntries(ctx context.Context, offset, limit int) ([]*domain.RiskListEntry, int64, error) {
	return s.riskListRepo.List(ctx, offset, limit)
}

// RemoveFromRiskList removes a social account from the risk list (admin action).
func (s *RiskService) RemoveFromRiskList(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	entry, err := s.riskListRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.riskListRepo.Delete(ctx, id); err != nil {
		return err
	}
	rt := "risk_list"
	rid := entry.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &adminID,
		Action:       "risk.list_removed",
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"provider": entry.Provider, "provider_uid": entry.ProviderUID, "user_id": entry.UserID, "reason": entry.Reason},
		CreatedAt:    time.Now().UTC(),
	})
	if entry.UserID != nil {
		return s.restoreUserIfRiskCleared(ctx, *entry.UserID)
	}
	return nil
}

func (s *RiskService) restoreUserIfRiskCleared(ctx context.Context, userID uuid.UUID) error {
	entries, err := s.riskListRepo.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list user risk entries: %w", err)
	}
	if len(entries) > 0 {
		return nil
	}
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if user.Status == domain.UserStatusSuspended {
		user.Status = domain.UserStatusActive
		if err := s.userRepo.Update(ctx, user); err != nil {
			return fmt.Errorf("restore user status: %w", err)
		}
		rt := "user"
		rid := userID.String()
		_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
			ID:           uuid.New(),
			UserID:       &userID,
			Action:       "risk.restored",
			ResourceType: &rt,
			ResourceID:   &rid,
			Details:      map[string]any{"reason": "risk_cleared"},
			CreatedAt:    time.Now().UTC(),
		})
	}
	if s.securitySvc != nil {
		if _, err := s.securitySvc.ComputeSecurityLevel(ctx, userID); err != nil {
			return fmt.Errorf("recompute security level: %w", err)
		}
	}
	return nil
}

// ListPendingReports returns reports waiting for admin review.
func (s *RiskService) ListPendingReports(ctx context.Context, offset, limit int) ([]*domain.RiskReport, int64, error) {
	return s.reportRepo.ListPending(ctx, offset, limit)
}

// ListReportsByTarget returns all reports for a specific user.
func (s *RiskService) ListReportsByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.RiskReport, error) {
	return s.reportRepo.ListByTarget(ctx, targetID)
}

// notifyReporter sends notification to the reporter about the report outcome.
// If SMTP is configured and user has email notifications enabled, send email.
// Otherwise, log to user's audit log.
func (s *RiskService) notifyReporter(ctx context.Context, report *domain.RiskReport, outcome, reason string) {
	reporter, err := s.userRepo.GetByID(ctx, report.ReporterID)
	if err != nil {
		return
	}

	// Check if SMTP is configured and user wants email notifications
	if s.emailSvc != nil && reporter.RiskReportEmailEnabled {
		// Try to send email
		if err := s.emailSvc.SendRiskReportResolved(ctx, reporter.Email, report.ID.String(), outcome, reason); err == nil {
			return
		}
		// If email fails, fall through to audit log
	}

	// Log to user's audit log
	action := "risk.report_confirmed_notification"
	if outcome == "dismissed" {
		action = "risk.report_dismissed_notification"
	}

	rt := "risk_report"
	rid := report.ID.String()
	_ = s.auditRepo.CreateLog(ctx, &domain.AuditLog{
		ID:           uuid.New(),
		UserID:       &report.ReporterID,
		Action:       action,
		ResourceType: &rt,
		ResourceID:   &rid,
		Details:      map[string]any{"outcome": outcome, "reason": reason, "target": report.TargetID.String()},
		CreatedAt:    time.Now().UTC(),
	})
}
