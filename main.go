package main

import (
	"os"

	"github.com/Gradient-Linux/concave-resolver/internal/resolver"
)

func main() {
	os.Exit(resolver.RunCLI(os.Args[1:], os.Stdout, os.Stderr))
}
