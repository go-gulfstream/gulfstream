package tests

import (
	"os"
	"testing"
)

const (
	RedisAddr    = "127.0.0.1:6380"
	PostgresAddr = "postgres://root:root@127.0.0.1:5435/postgres"
	KafkaAddr    = ""
)

func SkipIfNotIntegration(t *testing.T) {
	if os.Getenv("GULFSTREAM_INTEGRATION") != "ok" {
		t.Skip()
	}
}
