package main

import "github.com/hlonglin/unifiedproxy/internal/app"

var version = "dev"

func main() {
	app.RunMode(version, "control")
}
