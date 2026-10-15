package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestCreateAndFindOperator(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	created := f.createOperator(t, "owner")
	if created.ID == 0 || created.TOTPEnabled || created.CreatedAt.IsZero() {
		t.Fatalf("created = %+v", created)
	}
	for name, find := range map[string]func() (Operator, error){
		"by username": func() (Operator, error) { return f.operators.ByUsername(ctx, "owner") },
		"by id":       func() (Operator, error) { return f.operators.ByID(ctx, created.ID) },
		"first":       func() (Operator, error) { return f.operators.First(ctx) },
	} {
		got, err := find()
		if err != nil || got != created {
			t.Errorf("%s = %+v, %v", name, got, err)
		}
	}
	if _, err := f.operators.ByUsername(ctx, "nobody"); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing user err = %v", err)
	}
}

func TestUsernamesAreUnique(t *testing.T) {
	f := newFixture(t)
	f.createOperator(t, "owner")
	other := f.createOperator(t, "second")
	if _, err := f.operators.Create(context.Background(), "owner", testHash); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("Create duplicate err = %v", err)
	}
	if err := f.operators.SetUsername(context.Background(), other.ID, "owner"); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("SetUsername duplicate err = %v", err)
	}
}

func TestUpdatesTouchOnlyTheirOperator(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.createOperator(t, "owner")
	if err := f.operators.SetUsername(ctx, o.ID, "renamed"); err != nil {
		t.Fatal(err)
	}
	if err := f.operators.SetPasswordHash(ctx, o.ID, "new-hash"); err != nil {
		t.Fatal(err)
	}
	got, _ := f.operators.ByID(ctx, o.ID)
	if got.Username != "renamed" || got.PasswordHash != "new-hash" {
		t.Fatalf("got %+v", got)
	}
	if err := f.operators.SetPasswordHash(ctx, 999, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update of missing operator err = %v", err)
	}
}

func TestTOTPLifecycleKeepsSecretSealed(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.createOperator(t, "owner")
	sec := []byte("0123456789abcdefghij")

	if _, err := f.operators.PendingTOTP(ctx, o.ID); !errors.Is(err, ErrNoPendingTOTP) {
		t.Fatalf("pending before setup err = %v", err)
	}
	if err := f.operators.SetPendingTOTP(ctx, o.ID, sec); err != nil {
		t.Fatal(err)
	}
	if err := f.operators.ActivateTOTP(ctx, o.ID, 100); err != nil {
		t.Fatal(err)
	}
	got, last, err := f.operators.TOTP(ctx, o.ID)
	if err != nil || !bytes.Equal(got, sec) || last != 100 {
		t.Fatalf("TOTP = %q, %d, %v", got, last, err)
	}
	if op, _ := f.operators.ByID(ctx, o.ID); !op.TOTPEnabled {
		t.Fatal("TOTPEnabled = false after activation")
	}

	var raw []byte
	if err := f.db.QueryRow(`SELECT totp_secret FROM operators WHERE id = ?`, o.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, sec) {
		t.Fatal("TOTP secret stored in plain text")
	}

	if err := f.operators.DisableTOTP(ctx, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.operators.TOTP(ctx, o.ID); err == nil {
		t.Fatal("TOTP still readable after disable")
	}
}

func TestUseTOTPStepOnlyMovesForward(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	o := f.createOperator(t, "owner")
	for _, tc := range []struct {
		step int64
		want bool
	}{{10, true}, {10, false}, {9, false}, {11, true}} {
		ok, err := f.operators.UseTOTPStep(ctx, o.ID, tc.step)
		if err != nil || ok != tc.want {
			t.Errorf("UseTOTPStep(%d) = %v, %v; want %v", tc.step, ok, err, tc.want)
		}
	}
}

func TestSealedTOTPCannotMoveToAnotherOperator(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	a, b := f.createOperator(t, "alpha"), f.createOperator(t, "bravo")
	if err := f.operators.SetPendingTOTP(ctx, a.ID, []byte("secret-of-alpha-0000")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE operators SET totp_pending = (SELECT totp_pending FROM operators WHERE id = ?) WHERE id = ?`, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.operators.PendingTOTP(ctx, b.ID); err == nil {
		t.Fatal("a sealed secret copied to another row opened")
	}
}
