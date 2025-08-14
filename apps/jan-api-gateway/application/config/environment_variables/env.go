package environment_variables

import (
	"fmt"
	"os"
	"reflect"
)

type EnvironmentVariable struct {
	JAN_INFERENCE_MODEL_URL string
	SERPER_API_KEY          string
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
			if v.Field(i).Kind() == reflect.String {
				v.Field(i).SetString(envValue)
			}
		}
	}
}

// Singleton
var EnvironmentVariables = EnvironmentVariable{}

// docker run --rm -it \
//   -p 8080:8080 \
//   -e JAN_INFERENCE_MODEL_URL="https://inference-dev.jan.ai/" \
//   -e SERPER_API_KEY="your-serper-key" \
//   jan-api-gateway:latest