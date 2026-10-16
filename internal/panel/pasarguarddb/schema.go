// Package pasarguarddb is the PasarGuard destination: it reads and writes a
// PasarGuard database directly, following the panel's own schema.
package pasarguarddb

// Revision is the alembic head of PasarGuard 5.4.1. Newer revisions are
// added one by one, each with its own fixture (owner decision 3).
const Revision = "48a6bcb8bba1"
