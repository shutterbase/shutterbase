package migrations

import "embed"

// FS holds the golang-migrate formatted SQL migration files, applied at boot.
//
//go:embed *.sql
var FS embed.FS
