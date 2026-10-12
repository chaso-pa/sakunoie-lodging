package services

import (
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

// TestMain drops this package's test databases when its tests finish.
func TestMain(m *testing.M) { testutil.Main(m) }
