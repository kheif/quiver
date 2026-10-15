package main

import (
	"os"

	"github.com/kheif/quiver/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
