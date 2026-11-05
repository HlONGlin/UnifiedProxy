package main

import "github.com/hlonglin/unifiedproxy/internal/app"

// version is replaced at build time with: -ldflags "-X main.version=<tag>".
// Keeping a useful fallback makes `go run .` and ad-hoc builds identifiable.
var version = "dev"

func main() {
	app.Run(version)
}
