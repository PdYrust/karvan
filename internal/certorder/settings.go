package certorder

import (
	"context"
	"errors"

	"github.com/PdYrust/karvan/internal/bridge"
	"github.com/PdYrust/karvan/internal/settings"
)

// SettingsView is GET /settings/acme: the settings, whether a test
// CA from the config replaces Let's Encrypt, and whether the directory
// in use has a registered account.
type SettingsView struct {
	settings.ACME
	TestDirectory bool `json:"testDirectory"`
	HasAccount    bool `json:"hasAccount"`
}

// GetSettings reads the Let's Encrypt settings.
func (s *Service) GetSettings(ctx context.Context) (SettingsView, error) {
	a, err := s.Settings.ACME(ctx)
	if err != nil {
		return SettingsView{}, err
	}
	dir, err := s.directory(ctx)
	if err != nil {
		return SettingsView{}, err
	}
	v := SettingsView{ACME: a, TestDirectory: s.TestDirectory != ""}
	err = s.DB.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM acme_accounts WHERE directory = ? AND url <> '' "+
		"AND key_sealed IS NOT NULL)", dir).Scan(&v.HasAccount)
	return v, err
}

// SetSettings stores the Let's Encrypt settings. Turning keepPort80 on
// opens the challenge port first and stores nothing when it cannot;
// staging changes new orders only.
func (s *Service) SetSettings(ctx context.Context, a settings.ACME) error {
	if err := a.Validate(); errors.Is(err, settings.ErrACMEEmail) {
		return &bridge.ACMEInvalidError{Field: "email"}
	}
	prev, err := s.Settings.ACME(ctx)
	if err != nil {
		return err
	}
	if err := s.Bridge.WantChallengePort(ctx, keepHolder, a.KeepPort80); err != nil {
		return err
	}
	if err := s.Settings.SetACME(ctx, a); err != nil {
		if a.KeepPort80 && !prev.KeepPort80 {
			s.release(keepHolder)
		}
		return err
	}
	return nil
}
