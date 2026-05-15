package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

type RiskService struct {
	reportRepo  port.RiskReportRepository
	riskListRepo port.RiskListRepository
	bindingRepo port.BindingRepository
	userRepo    port.UserRepository
	auditRepo   port.AuditRepository
	securitySvc *SecurityLevelService
}

func NewRiskService(
	reportRepo port.RiskReportRepository,
	riskListRepo port.RiskListRepository,
	bindingRepo port.BindingRepository,
	userRepo port.UserRepository,
	auditRepo port.AuditRepository,
	securitySvc *SecurityLevelService,
) *RiskService {
	return &RiskService{
		reportRepo:   reportRepo,
		riskListRepo: riskListRepo,
		bindingRepo:  bindingRepo,
		userRepo:     userRepo,
		auditRepo:    auditRepo,
		securitySvc:  securitySvc,
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

	// Auto-escalate: if user has >= 3 confirmed reports, auto-downgrade.
	confirmed, _ := s.reportRepo.CountConfirmedByTarget(ctx, targetID)
	if confirmed >= 2 { // This is the 3rd+ report (current one still pending counts toward urgency)
		// Auto-confirm this report and trigger enforcement.
		report.Status = domain.ReportStatusConfirmed
		now := time.Now().UTC()
		report.ResolvedAt = &now
		_ = s.reportRepo.Update(ctx, report)
		_ = s.enforceRisk(ctx, targetID, report.ID)
	}

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

	return s.enforceRisk(ctx, report.TargetID, report.ID)
}

// DismissReport marks a report as dismissed (false positive).
func (s *RiskService) DismissReport(ctx context.Context, reportID, adminID uuid.UUID, note string) error {
	report, err := s.reportRepo.GetByID(ctx, reportID)
	if err != nil {
		return fmt.Errorf("get report: %w", err)
	}
	now := time.Now().UTC()
	report.Status = domain.ReportStatusDismissed
	report.AdminNote = note
	report.ResolvedAt = &now
	report.ResolvedBy = &adminID
	return s.reportRepo.Update(ctx, report)
}

// enforceRisk lowers the user's trust level to 0, suspends the account, and adds their social bindings to the risk list.
func (s *RiskService) enforceRisk(ctx context.Context, userID uuid.UUID, reportID uuid.UUID) error {
	// Lower trust level to 0.
	if err := s.userRepo.UpdateSecurityLevel(ctx, userID, 0); err != nil {
		return fmt.Errorf("downgrade user: %w", err)
	}

	// Suspend the user account to block password login as well.
	user, err := s.userRepo.GetByID(ctx, userID)
	if err == nil && user.Status == domain.UserStatusActive {
		user.Status = domain.UserStatusSuspended
		_ = s.userRepo.Update(ctx, user)
	}

	_ = s.auditRepo.CreateSecurityLevelChange(ctx, &domain.SecurityLevelChange{
		ID:        uuid.New(),
		UserID:    userID,
		OldLevel:  -1, // Unknown, just marking as risk enforcement
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

// ListRiskEntries returns the risk blacklist.
func (s *RiskService) ListRiskEntries(ctx context.Context, offset, limit int) ([]*domain.RiskListEntry, int64, error) {
	return s.riskListRepo.List(ctx, offset, limit)
}

// RemoveFromRiskList removes a social account from the risk list (admin action).
func (s *RiskService) RemoveFromRiskList(ctx context.Context, id uuid.UUID) error {
	return s.riskListRepo.Delete(ctx, id)
}

// ListPendingReports returns reports waiting for admin review.
func (s *RiskService) ListPendingReports(ctx context.Context, offset, limit int) ([]*domain.RiskReport, int64, error) {
	return s.reportRepo.ListPending(ctx, offset, limit)
}

// ListReportsByTarget returns all reports for a specific user.
func (s *RiskService) ListReportsByTarget(ctx context.Context, targetID uuid.UUID) ([]*domain.RiskReport, error) {
	return s.reportRepo.ListByTarget(ctx, targetID)
}
