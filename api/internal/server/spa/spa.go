// Package spa embeds the built Quasar SPA so the server ships as a single
// self-contained binary (the whole point of the rewrite vs. the old
// directory-served deploy).
//
// In dev/CI the dist/ directory holds only a placeholder index.html so
// `go build`/`go test` compile. The Docker build overwrites dist/ with the
// real `bun run build` output BEFORE `go build`, so the production image
// embeds the actual app. ponytail: embed over a served directory — one binary,
// no web/ dir to ship alongside.
package spa

import "embed"

//go:embed all:dist
var FS embed.FS
