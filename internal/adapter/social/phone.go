package social

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/anthropic/oidc-platform/internal/domain"
	"github.com/anthropic/oidc-platform/internal/port"
)

// PhoneCodeVerifier verifies an SMS code for the given phone number. It must
// return nil only when the (phone, code) pair matches an unexpired, unused
// verification request. The Phone provider does not generate or send codes —
// the service layer is responsible for that and provides the verifier.
type PhoneCodeVerifier interface {
	VerifyPhoneCode(ctx context.Context, phoneNumber, code string) error
}

type PhoneProvider struct {
	verifier PhoneCodeVerifier
}

func NewPhoneProvider(verifier PhoneCodeVerifier) *PhoneProvider {
	return &PhoneProvider{verifier: verifier}
}

func (p *PhoneProvider) Name() string { return domain.ProviderPhone }

func (p *PhoneProvider) BeginAuth(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

type phoneCallbackRequest struct {
	PhoneNumber string `json:"phone_number"`
	Code        string `json:"code"`
}

func (p *PhoneProvider) CompleteAuth(ctx context.Context, r *http.Request) (*port.ProviderUserInfo, error) {
	var req phoneCallbackRequest

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, fmt.Errorf("decode phone request: %w", err)
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("parse form: %w", err)
		}
		req.PhoneNumber = r.FormValue("phone_number")
		req.Code = r.FormValue("code")
	}

	req.PhoneNumber = strings.TrimSpace(req.PhoneNumber)
	req.Code = strings.TrimSpace(req.Code)

	if req.PhoneNumber == "" {
		return nil, fmt.Errorf("missing phone_number")
	}
	if req.Code == "" {
		return nil, fmt.Errorf("missing code")
	}

	if p.verifier == nil {
		return nil, fmt.Errorf("phone provider has no verifier configured")
	}
	if err := p.verifier.VerifyPhoneCode(ctx, req.PhoneNumber, req.Code); err != nil {
		return nil, fmt.Errorf("verify phone code: %w", err)
	}

	return &port.ProviderUserInfo{
		ProviderUID: req.PhoneNumber,
		DisplayName: req.PhoneNumber,
		RawProfile: map[string]any{
			"phone_number": req.PhoneNumber,
		},
	}, nil
}

func (p *PhoneProvider) SupportsRefresh() bool { return false }

func (p *PhoneProvider) RefreshToken(_ context.Context, _ string) (*port.ProviderTokenInfo, error) {
	return nil, fmt.Errorf("phone provider does not support refresh")
}

var _ port.SocialProvider = (*PhoneProvider)(nil)
