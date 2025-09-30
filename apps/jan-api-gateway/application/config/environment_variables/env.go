package environment_variables

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"menlo.ai/jan-api-gateway/app/utils/logger"
	"menlo.ai/jan-api-gateway/config"
)

type EnvironmentVariable struct {
	JAN_INFERENCE_MODEL_URL     string
	SERPER_API_KEY              string
	JWT_SECRET                  []byte
	OAUTH2_GOOGLE_CLIENT_ID     string
	OAUTH2_GOOGLE_CLIENT_SECRET string
	OAUTH2_GOOGLE_REDIRECT_URL  string
	DB_POSTGRESQL_WRITE_DSN     string
	DB_POSTGRESQL_READ1_DSN     string
	APIKEY_SECRET               string
	MODEL_PROVIDER_SECRET       string
	ALLOWED_CORS_HOSTS          []string
	SMTP_HOST                   string
	SMTP_PORT                   int
	SMTP_USERNAME               string
	SMTP_PASSWORD               string
	SMTP_SENDER_EMAIL           string
	INVITE_REDIRECT_URL         string
	ORGANIZATION_ADMIN_EMAILS   []string
	// Redis configuration
	REDIS_URL      string
	REDIS_PASSWORD string
	REDIS_DB       int

	// inference provider
	OPENROUTER_BASE_URL string
	GEMINI_BASE_URL     string
}

func (ev *EnvironmentVariable) LoadFromEnv() {
	v := reflect.ValueOf(ev).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		envKey := field.Name
		envValue := os.Getenv(envKey)
		if envValue == "" {
			logger.GetLogger().Warnf("Missing SYSENV: %s", envKey)
		}
		if envValue != "" {
			switch v.Field(i).Kind() {
			case reflect.String:
				v.Field(i).SetString(envValue)
			case reflect.Int:
				intV, err := strconv.Atoi(envValue)
				if err != nil {
					logger.GetLogger().Errorf("Invalid int value for %s: %s", envKey, envValue)
				} else {
					v.Field(i).SetInt(int64(intV))
				}
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(envValue)
				if err != nil {
					logger.GetLogger().Errorf("Invalid boolean value for %s: %s", envKey, envValue)
				} else {
					v.Field(i).SetBool(boolVal)
				}
			case reflect.Slice:
				if v.Field(i).Type().Elem().Kind() == reflect.Uint8 {
					v.Field(i).SetBytes([]byte(envValue))
				} else if v.Field(i).Type().Elem().Kind() == reflect.String {
					entries := strings.Split(envValue, ",")
					v.Field(i).Set(reflect.ValueOf(entries))
				} else {
					logger.GetLogger().Errorf("Unsupported slice type for %s", field.Name)
				}
			default:
				logger.GetLogger().Errorf("Unsupported field type: %s", field.Name)
			}
		}
	}
	config.EnvReloadedAt = time.Now()
}

// Singleton
var EnvironmentVariables = EnvironmentVariable{}
