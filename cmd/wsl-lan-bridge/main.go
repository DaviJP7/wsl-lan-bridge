package main

import (
	"os"

	"github.com/david/wsl-lan-bridge/internal/app"
)

func main() {
	os.Exit(app.Run(os.Args[1:]))
}
