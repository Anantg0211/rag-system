package migrations

import "embed"

// FS contains the versioned database migrations applied at application startup.
//
//go:embed *.sql
var FS embed.FS
