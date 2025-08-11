package environment_variables

import (
	"fmt"
	"os"
	"reflect"
)

type EnvironmentVariable struct {
	JAN_INFERENCE_MODEL_URL string
}

func (ev *EnvironmentVariable) LoadFromEnv() {
	v := reflect.ValueOf(ev).Elem()
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		envKey := field.Name
		envValue := os.Getenv(envKey)
		if envValue == "" {
			fmt.Printf("Missing SYSENV: %s", envKey)
		}
		if envValue != "" {
			if v.Field(i).Kind() == reflect.String {
				v.Field(i).SetString(envValue)
			}
		}
	}
}

// Singleton
var EnvironmentVariables = EnvironmentVariable{}
