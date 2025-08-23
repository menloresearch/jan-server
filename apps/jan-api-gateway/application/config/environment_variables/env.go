package environment_variables

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type EnvironmentVariable struct {
	JAN_INFERENCE_MODEL_URL     string
	SERPER_API_KEY              string
	ENABLE_ADMIN_API            bool
	JWT_SECRET                  []byte
	OAUTH2_GOOGLE_CLIENT_ID     string
	OAUTH2_GOOGLE_CLIENT_SECRET string
	OAUTH2_GOOGLE_REDIRECT_URL  string
	DB_POSTGRESQL_WRITE_DSN     string
	DB_POSTGRESQL_READ1_DSN     string
	APIKEY_SECRET               string
}

func (ev *EnvironmentVariable) LoadFromEnv() {
	v := reflect.ValueOf(ev).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		envKey := field.Name
		envValue := os.Getenv(envKey)
		if envValue == "" {
			fmt.Printf("Missing SYSENV: %s\n", envKey)
		}
		if envValue != "" {
			switch v.Field(i).Kind() {
			case reflect.String:
				v.Field(i).SetString(envValue)
			case reflect.Bool:
				boolVal, err := strconv.ParseBool(envValue)
				if err != nil {
					fmt.Printf("Invalid boolean value for %s: %s\n", envKey, envValue)
				} else {
					v.Field(i).SetBool(boolVal)
				}
			case reflect.Slice:
				if v.Field(i).Type().Elem().Kind() == reflect.Uint8 {
					v.Field(i).SetBytes([]byte(envValue))
				} else {
					fmt.Printf("Unsupported slice type for %s\n", field.Name)
				}
			default:
				fmt.Printf("Unsupported field type: %s\n", field.Name)
			}
		}
	}
}

// Singleton
var EnvironmentVariables = EnvironmentVariable{}
