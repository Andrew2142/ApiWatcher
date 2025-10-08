package email

import (
	"net/smtp"
	"url-checker/internal/config"
)

// ==========================
// Email Alert Function
// ==========================
func Send(to, subject, body string) error {
	addr := config.SMTPHost + ":" + config.SMTPPort

	msg := []byte("To: " + to + "\r\n" +
	"Subject: " + subject + "\r\n\r\n" +
	body + "\r\n")

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPass, config.SMTPHost)
	return smtp.SendMail(addr, auth, config.SMTPFrom, []string{to}, msg)
}

