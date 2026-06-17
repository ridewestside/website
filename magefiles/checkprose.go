//go:build mage

package main

import (
	"fmt"
	"os/exec"

	"github.com/magefile/mage/sh"
)

// CheckProse runs Vale prose linting on all content files
func CheckProse() error {
	if _, err := exec.LookPath("vale"); err != nil {
		return fmt.Errorf("vale not found — run: mise install vale\n  or see https://vale.sh/docs/vale-cli/installation/")
	}
	fmt.Println("Checking prose...")
	return sh.RunV("vale", "content/")
}
