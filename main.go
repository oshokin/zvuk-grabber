/*
Copyright © 2025 Oleg Shokin

This file is the entry point for the zvuk-grabber application.
It initializes and executes the root command defined in the cmd package.
*/
package main

import (
	"fmt"
	"os"

	"github.com/oshokin/zvuk-grabber/cmd"
)

// main is the entry point of the application.
// It calls the Execute function from the cmd package, which starts the CLI.
func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
