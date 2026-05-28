package smtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/anthropic/oidc-platform/internal/config"
	"github.com/anthropic/oidc-platform/internal/port"
)

// Sender implements port.EmailSender using SMTP.
// It reads configuration from the settings database at send time, falling back to config file values.
type Sender struct {
	fallbackHost     string
	fallbackPort     int
	fallbackUsername string
	fallbackPassword string
	fallbackFrom     string
	baseURL          string
	settingsRepo     port.SettingsRepository
}

func NewSender(cfg config.SMTPConfig, baseURL string, settingsRepo port.SettingsRepository) *Sender {
	return &Sender{
		fallbackHost:     cfg.Host,
		fallbackPort:     cfg.Port,
		fallbackUsername: cfg.Username,
		fallbackPassword: cfg.Password,
		fallbackFrom:     cfg.From,
		baseURL:          strings.TrimRight(baseURL, "/"),
		settingsRepo:     settingsRepo,
	}
}

func (s *Sender) getConfig(ctx context.Context) (host string, port int, username, password, from string) {
	host = s.fallbackHost
	port = s.fallbackPort
	username = s.fallbackUsername
	password = s.fallbackPassword
	from = s.fallbackFrom

	if s.settingsRepo == nil {
		return
	}
	if v, err := s.settingsRepo.Get(ctx, "smtp_host"); err == nil && v.Value != "" {
		host = v.Value
	}
	if v, err := s.settingsRepo.Get(ctx, "smtp_port"); err == nil && v.Value != "" {
		if p, err := strconv.Atoi(v.Value); err == nil {
			port = p
		}
	}
	if v, err := s.settingsRepo.Get(ctx, "smtp_username"); err == nil && v.Value != "" {
		username = v.Value
	}
	if v, err := s.settingsRepo.Get(ctx, "smtp_password"); err == nil && v.Value != "" {
		password = v.Value
	}
	if v, err := s.settingsRepo.Get(ctx, "smtp_from"); err == nil && v.Value != "" {
		from = v.Value
	}
	return
}

func (s *Sender) SendRegistrationCode(ctx context.Context, to, code string) error {
	subject := "注册验证码 / Registration code"
	body := fmt.Sprintf(`您好，

您的注册验证码是：%s

验证码 10 分钟内有效。验证通过后才会创建账户。如果您没有注册账户，请忽略此邮件。

---
Hello,

Your registration code is: %s

This code expires in 10 minutes. Your account will be created only after verification. If you didn't request this, please ignore this email.
`, code, code)

	return s.send(ctx, to, subject, body)
}

func (s *Sender) SendVerificationEmail(ctx context.Context, to, token string) error {
	link := fmt.Sprintf("%s/verify-email?token=%s", s.baseURL, token)
	subject := "验证您的邮箱 / Verify your email"
	body := fmt.Sprintf(`您好，

请点击以下链接验证您的邮箱地址：

%s

此链接 24 小时内有效。如果您没有注册账户，请忽略此邮件。

---
Hello,

Please click the link below to verify your email address:

%s

This link expires in 24 hours. If you didn't create an account, please ignore this email.
`, link, link)

	return s.send(ctx, to, subject, body)
}

func (s *Sender) SendPasswordResetEmail(ctx context.Context, to, token string) error {
	link := fmt.Sprintf("%s/reset-password?token=%s", s.baseURL, token)
	subject := "重置密码 / Reset your password"
	body := fmt.Sprintf(`您好，

您请求了密码重置。请点击以下链接设置新密码：

%s

此链接 1 小时内有效。如果您没有请求重置密码，请忽略此邮件。

---
Hello,

You requested a password reset. Click the link below to set a new password:

%s

This link expires in 1 hour. If you didn't request this, please ignore this email.
`, link, link)

	return s.send(ctx, to, subject, body)
}

func (s *Sender) SendRiskReportResolved(ctx context.Context, to, reportID, outcome, reason string) error {
	var subject, body string

	if outcome == "confirmed" {
		subject = "举报已确认 / Report Confirmed"
		if reason != "" {
			body = fmt.Sprintf(`您好，

您提交的举报（ID: %s）已被管理员确认。

管理员备注：%s

感谢您帮助维护平台安全。

---
Hello,

Your report (ID: %s) has been confirmed by an administrator.

Admin note: %s

Thank you for helping keep the platform safe.
`, reportID, reason, reportID, reason)
		} else {
			body = fmt.Sprintf(`您好，

您提交的举报（ID: %s）已被管理员确认。

感谢您帮助维护平台安全。

---
Hello,

Your report (ID: %s) has been confirmed by an administrator.

Thank you for helping keep the platform safe.
`, reportID, reportID)
		}
	} else {
		subject = "举报已驳回 / Report Dismissed"
		if reason != "" {
			body = fmt.Sprintf(`您好，

您提交的举报（ID: %s）已被管理员驳回。

驳回原因：%s

如有疑问，请联系管理员。

---
Hello,

Your report (ID: %s) has been dismissed by an administrator.

Reason: %s

If you have questions, please contact an administrator.
`, reportID, reason, reportID, reason)
		} else {
			body = fmt.Sprintf(`您好，

您提交的举报（ID: %s）已被管理员驳回。

如有疑问，请联系管理员。

---
Hello,

Your report (ID: %s) has been dismissed by an administrator.

If you have questions, please contact an administrator.
`, reportID, reportID)
		}
	}

	return s.send(ctx, to, subject, body)
}

func (s *Sender) send(ctx context.Context, to, subject, body string) error {
	host, port, username, password, from := s.getConfig(ctx)

	if host == "" {
		slog.Warn("SMTP not configured, skipping email", "to", to, "subject", subject)
		return nil
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, body)

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	var auth smtp.Auth
	if username != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	if port == 465 {
		return sendTLS(addr, host, auth, from, to, []byte(msg))
	}
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}

func sendTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
