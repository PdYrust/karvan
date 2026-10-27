package discovery

import (
	"path/filepath"
	"slices"
)

// CertEnv is what the certificate scan needs from one panel ENV file: the
// panel's folder name (as in /var/lib/<Folder>/certs) and the certificate
// paths the panel serves, as written.
type CertEnv struct {
	Folder   string
	CertFile string // UVICORN_SSL_CERTFILE
	KeyFile  string // UVICORN_SSL_KEYFILE
}

// CertEnvs reads the certificate paths of the panel ENV files the panel
// scan reads (<root>/*/.env and extra). It probes no database. A file that
// cannot be read, or is not a panel's, gives nothing.
func CertEnvs(roots, extra []string) []CertEnv {
	var out []CertEnv
	for _, e := range envFiles(roots, extra) {
		env, err := readEnv(e)
		if err != nil {
			continue
		}
		dir := filepath.Dir(e)
		if env["SQLALCHEMY_DATABASE_URL"] == "" && composeHint(dir) == "" {
			continue
		}
		out = append(out, CertEnv{Folder: filepath.Base(dir), CertFile: env["UVICORN_SSL_CERTFILE"],
			KeyFile: env["UVICORN_SSL_KEYFILE"]})
	}
	return out
}

// envFiles are every <root>/*/.env and the extra ENV files, sorted, each
// once.
func envFiles(roots, extra []string) []string {
	var envs []string
	for _, r := range roots {
		matches, _ := filepath.Glob(filepath.Join(r, "*", ".env"))
		envs = append(envs, matches...)
	}
	envs = append(envs, extra...)
	slices.Sort(envs)
	return slices.Compact(envs)
}
