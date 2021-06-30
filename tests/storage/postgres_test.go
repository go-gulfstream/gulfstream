package storage

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/go-gulfstream/gulfstream/tests"
)

func TestStorage_Postgres(t *testing.T) {
	tests.SkipIfNotIntegration(t)
}

type PostgresSuite struct {
	suite.Suite
}
