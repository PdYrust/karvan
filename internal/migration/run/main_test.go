package run

import (
	"os"
	"testing"
)

// startHashCost is the bcrypt cost runs had when the test binary started,
// before TestMain lowered it.
var startHashCost int

func TestMain(m *testing.M) {
	startHashCost = hashCost
	UseCheapHashesForTests()
	os.Exit(m.Run())
}

// useHashCost sets the bcrypt cost of runs for one test.
func useHashCost(t *testing.T, cost int) {
	t.Helper()
	before := hashCost
	hashCost = cost
	t.Cleanup(func() { hashCost = before })
}
