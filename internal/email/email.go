package email

import (
	"apiwatcher/internal/config"
	"fmt"
	"net/smtp"
)

// ==========================
// Email Alert Function
// ==========================
func Send(to, subject, body string) error {
	// Try to load SMTP config from file first
	smtpConfig, err := config.LoadSMTPConfig()
	if err != nil || smtpConfig == nil {
		// Fall back to environment variables (legacy support)
		return sendWithEnvVars(to, subject, body)
	}

	// Use SMTP config from file
	addr := smtpConfig.Host + ":" + smtpConfig.Port

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	return smtp.SendMail(addr, auth, smtpConfig.From, []string{to}, msg)
}

// sendWithEnvVars sends email using environment variables (legacy support)
func sendWithEnvVars(to, subject, body string) error {
	if config.SMTPHost == "" || config.SMTPPort == "" {
		return fmt.Errorf("SMTP not configured - please set up SMTP in the GUI")
	}

	addr := config.SMTPHost + ":" + config.SMTPPort

	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPass, config.SMTPHost)
	return smtp.SendMail(addr, auth, config.SMTPFrom, []string{to}, msg)
}
