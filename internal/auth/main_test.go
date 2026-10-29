package auth

import (
	"os"
	"testing"
)

// startArgon is the cost HashPassword had when the test binary started,
// before TestMain lowered it.
var startArgon argonParams

func TestMain(m *testing.M) {
	startArgon = argon
	UseCheapHashesForTests()
	os.Exit(m.Run())
}

// useArgon sets the cost of new hashes for one test.
func useArgon(t *testing.T, p argonParams) {
	t.Helper()
	before := argon
	argon = p
	t.Cleanup(func() { argon = before })
}
