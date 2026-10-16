package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/netip"
	"time"
)

// LockoutPolicy sets how many failed sign-ins lock an IP or an account.
type LockoutPolicy struct {
	IPMaxFailures int
	IPWindow      time.Duration
	IPLock        time.Duration

	AccountMaxFailures int
	AccountWindow      time.Duration
	// AccountLocks grow with each lock in a row; the last one repeats.
	AccountLocks []time.Duration
	// A quiet period this long resets the account lock level.
	AccountForgive time.Duration
}

// DefaultLockout is the policy from the phase 2 spec (D10).
var DefaultLockout = LockoutPolicy{
	IPMaxFailures:      5,
	IPWindow:           15 * time.Minute,
	IPLock:             15 * time.Minute,
	AccountMaxFailures: 10,
	AccountWindow:      15 * time.Minute,
	AccountLocks:       []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour},
	AccountForgive:     24 * time.Hour,
}

const (
	scopeIP      = "ip"
	scopeAccount = "account"
)

// Limiter counts failed sign-ins in the database, so a restart does not
// reset them.
type Limiter struct {
	db     *sql.DB
	now    func() time.Time
	policy LockoutPolicy
}

// NewLimiter returns a Limiter with policy.
func NewLimiter(db *sql.DB, policy LockoutPolicy) *Limiter {
	return &Limiter{db: db, now: time.Now, policy: policy}
}

// ipKey groups IPv6 clients by /64, the block one subscriber usually gets,
// so rotating through addresses inside it does not reset the counter.
func ipKey(ip string) string {
	a, err := netip.ParseAddr(ip)
	if err != nil || !a.Unmap().Is6() {
		return ip
	}
	p, _ := a.Prefix(64)
	return p.String()
}

// Check returns how long the IP or account must still wait; 0 means go ahead.
func (l *Limiter) Check(ctx context.Context, ip, account string) (time.Duration, error) {
	ip = ipKey(ip)
	var until int64
	err := l.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(locked_until), 0) FROM login_failures
		WHERE (scope = ? AND key = ?) OR (scope = ? AND key = ?)`, scopeIP, ip, scopeAccount, account).Scan(&until)
	if err != nil {
		return 0, fmt.Errorf("check lockout: %w", err)
	}
	return max(time.Unix(until, 0).Sub(l.now()), 0), nil
}

// Fail records a failed sign-in. It returns the lock that started because of
// this failure, or 0 when none did.
func (l *Limiter) Fail(ctx context.Context, ip, account string) (time.Duration, error) {
	ip = ipKey(ip)
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("record failed sign-in: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ipLock, err := l.bump(ctx, tx, scopeIP, ip)
	if err != nil {
		return 0, err
	}
	accountLock, err := l.bump(ctx, tx, scopeAccount, account)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("record failed sign-in: %w", err)
	}
	return max(ipLock, accountLock), nil
}

// Succeed clears the counters after a correct sign-in.
func (l *Limiter) Succeed(ctx context.Context, ip, account string) error {
	ip = ipKey(ip)
	_, err := l.db.ExecContext(ctx, `DELETE FROM login_failures WHERE (scope = ? AND key = ?) OR (scope = ? AND key = ?)`,
		scopeIP, ip, scopeAccount, account)
	if err != nil {
		return fmt.Errorf("clear failed sign-ins: %w", err)
	}
	return nil
}

type failureRow struct {
	failures, windowStart, lockedUntil, level int64
}

func (l *Limiter) bump(ctx context.Context, tx *sql.Tx, scope, key string) (time.Duration, error) {
	now := l.now()
	var r failureRow
	err := tx.QueryRowContext(ctx, `SELECT failures, window_start, locked_until, lock_level FROM login_failures
		WHERE scope = ? AND key = ?`, scope, key).Scan(&r.failures, &r.windowStart, &r.lockedUntil, &r.level)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("read failed sign-ins: %w", err)
	}

	maxFailures, window := l.policy.IPMaxFailures, l.policy.IPWindow
	if scope == scopeAccount {
		maxFailures, window = l.policy.AccountMaxFailures, l.policy.AccountWindow
		if r.level > 0 && now.Sub(time.Unix(r.windowStart, 0)) > l.policy.AccountForgive {
			r.level = 0
		}
	}
	if now.Sub(time.Unix(r.windowStart, 0)) > window {
		r.failures, r.windowStart = 0, now.Unix()
	}
	r.failures++

	var lock time.Duration
	if r.failures >= int64(maxFailures) {
		lock = l.lockFor(scope, r.level)
		r.failures, r.windowStart, r.lockedUntil = 0, now.Unix(), now.Add(lock).Unix()
		r.level++
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO login_failures (scope, key, failures, window_start, locked_until, lock_level)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (scope, key) DO UPDATE SET failures = excluded.failures,
		window_start = excluded.window_start, locked_until = excluded.locked_until, lock_level = excluded.lock_level`,
		scope, key, r.failures, r.windowStart, r.lockedUntil, r.level)
	if err != nil {
		return 0, fmt.Errorf("save failed sign-ins: %w", err)
	}
	return lock, nil
}

func (l *Limiter) lockFor(scope string, level int64) time.Duration {
	if scope == scopeIP {
		return l.policy.IPLock
	}
	locks := l.policy.AccountLocks
	return locks[min(int(level), len(locks)-1)]
}

// Prune removes rows that no longer lock anything and have aged out.
func (l *Limiter) Prune(ctx context.Context) error {
	now := l.now()
	_, err := l.db.ExecContext(ctx, `DELETE FROM login_failures WHERE locked_until < ? AND window_start < ?`,
		now.Unix(), now.Add(-l.policy.AccountForgive).Unix())
	if err != nil {
		return fmt.Errorf("prune failed sign-ins: %w", err)
	}
	return nil
}
