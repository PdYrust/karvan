package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// The update, rollback and uninstall work is in the installer's scripts
// (D22). These commands only check the script and hand over to it.
var (
	lifecycleDir = "/usr/local/lib/karvan/current"
	geteuid      = os.Geteuid
	// scriptOwner is the owner a script must have: root.
	scriptOwner = 0
	execScript  = func(path string, args []string) error {
		return syscall.Exec(path, append([]string{path}, args...), scriptEnv()) //nolint:gosec // G204: a fixed, root-owned script checked by runScript.
	}
)

func runUpdate(_ context.Context, streams IO, args []string) error {
	fs := newFlagSet("update", streams)
	archive := fs.String("archive", "", "update from this release archive instead of downloading it")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	script := []string{"--update"}
	if *archive != "" {
		abs, err := filepath.Abs(*archive)
		if err != nil {
			return err
		}
		if fi, err := os.Stat(abs); err != nil || !fi.Mode().IsRegular() {
			return fmt.Errorf("%w: %s is not a file", errUsage, *archive)
		}
		script = append(script, "--archive", abs)
	}
	return runScript("get.sh", script)
}

func runRollback(_ context.Context, streams IO, args []string) error {
	fs := newFlagSet("rollback", streams)
	yes := fs.Bool("yes", false, "restore the backup without asking when the store is too new for the previous version")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	script := []string{"--rollback"}
	if *yes {
		script = append(script, "--yes")
	}
	return runScript("install.sh", script)
}

func runUninstall(_ context.Context, streams IO, args []string) error {
	fs := newFlagSet("uninstall", streams)
	purge := fs.Bool("purge", false, "also delete the data, the config and the backups (asks you to type delete karvan data)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	script := []string{"--uninstall"}
	if *purge {
		script = append(script, "--purge")
	}
	return runScript("install.sh", script)
}

// runScript runs the installer's script name with args in place of this
// process, after checking that only root can have written it.
func runScript(name string, args []string) error {
	if geteuid() != 0 {
		return errors.New("run it as root, for example with sudo")
	}
	path, err := filepath.EvalSymlinks(filepath.Join(lifecycleDir, name))
	if err != nil {
		return fmt.Errorf("the installer's %s is missing (was Karvan installed with install.sh?): %w", name, err)
	}
	if err := ownedByRoot(path, false); err != nil {
		return err
	}
	if err := ownedByRoot(filepath.Dir(path), true); err != nil {
		return err
	}
	return execScript(path, args)
}

// ownedByRoot refuses path unless it is a file (or a folder, when dir)
// owned by root that neither its group nor others can write.
func ownedByRoot(path string, dir bool) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	switch {
	case dir && !fi.IsDir(), !dir && !fi.Mode().IsRegular():
		return fmt.Errorf("refusing %s: not a regular file or folder", path)
	case fi.Mode().Perm()&0o022 != 0:
		return fmt.Errorf("refusing %s: its group or others can write it (mode %s)", path, fi.Mode().Perm())
	}
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok || int(st.Uid) != scriptOwner {
		return fmt.Errorf("refusing %s: not owned by root", path)
	}
	return nil
}

// scriptEnv is a fixed environment for the script, so a PATH from the
// caller cannot swap the tools it runs as root.
func scriptEnv() []string {
	env := []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	for _, k := range []string{"TERM", "LANG", "NO_COLOR"} {
		if v, ok := os.LookupEnv(k); ok {
			env = append(env, k+"="+v)
		}
	}
	return env
}
