// Package migrations embeds goose migration files for use as a library.
package migrations

import "embed"

// FS holds the embedded SQL migration files.
//
//go:embed *.sql
var FS embed.FS
