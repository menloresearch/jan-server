package emailservice

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func SendEmail(to string, subject string, body string) error {
	envs := environment_variables.EnvironmentVariables

	addr := fmt.Sprintf("%s:%d", envs.SMTP_HOST, envs.SMTP_PORT)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         envs.SMTP_HOST,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS dial error: %w", err)
	}

	client, err := smtp.NewClient(conn, envs.SMTP_HOST)
	if err != nil {
		return fmt.Errorf("SMTP client error: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth("", envs.SMTP_USERNAME, envs.SMTP_PASSWORD, envs.SMTP_HOST)
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP auth error: %w", err)
	}

	headers := ""
	headers += "MIME-Version: 1.0\r\n"
	headers += "Content-Type: text/html; charset=\"UTF-8\"\r\n"
	headers += "From: " + envs.SMTP_SENDER_EMAIL + "\r\n"
	headers += "To: " + to + "\r\n"
	headers += "Subject: " + subject + "\r\n"

	msg := headers + "\r\n" + body

	if err = client.Mail(envs.SMTP_SENDER_EMAIL); err != nil {
		return err
	}

	if err = client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(msg))
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}
