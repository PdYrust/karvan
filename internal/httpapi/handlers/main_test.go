package handlers_test

import (
	"os"
	"testing"

	"github.com/PdYrust/karvan/internal/auth"
	"github.com/PdYrust/karvan/internal/migration/run"
)

// The auth and run tests check the production hash costs; here every
// sign-in and every run would otherwise spend most of the time hashing.
func TestMain(m *testing.M) {
	auth.UseCheapHashesForTests()
	run.UseCheapHashesForTests()
	os.Exit(m.Run())
}
