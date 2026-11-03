package main

import "github.com/RomanovCaesar/m-ui/internal/app"

var version = "dev"

func main() {
	app.RunMode(version, "control")
}
