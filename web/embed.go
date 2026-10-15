// Package web embeds the built web panel into the karvan binary.
package web

import (
	"embed"
	"fmt"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist returns the build output (the contents of web/dist).
func Dist() (fs.FS, error) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded web build: %w", err)
	}
	return sub, nil
}
