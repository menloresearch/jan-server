package config

import (
	"strings"
	"time"
)

var Version = "dev"
var EnvReloadedAt = time.Now()

func IsDev() bool {
	return strings.HasPrefix(Version, "dev")
}
