package emailservice

import (
	"net/smtp"

	"menlo.ai/jan-api-gateway/config/environment_variables"
)

func SendEmail(from string, to string, subject string, body string) {
	envs := environment_variables.EnvironmentVariables
	auth := smtp.PlainAuth(
		"", envs.SMTP_USERNAME, envs.SMTP_PASSWORD, envs.SMTP_HOST,
	)
	
}
