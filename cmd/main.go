/*
Copyright 2026.

Licensed under the European Union Public Licence (the ‘EUPL’), Version 1.2 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at: https://eupl.eu/1.2/en/
*/

package main

import (
	"os"
	"strings"
	"syscall"
)

// Redirect to ground-control binary for backwards compatibility
func main() {
	os.Args[0] = "ground-control"
	if before, ok := strings.CutSuffix(os.Args[0], ".test"); ok {
		os.Args[0] = before
	}
	_ = syscall.Exec(os.Args[0], os.Args, os.Environ())
}
