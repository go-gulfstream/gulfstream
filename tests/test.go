package tests

import (
	"os"
	"testing"
)

func SkipIfNotIntegration(t *testing.T) {
	if os.Getenv("GULFSTREAM_INTEGRATION") != "ok" {
		t.Skip()
	}
}
