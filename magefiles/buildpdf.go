//go:build mage

package main

import "os/exec"

// BuildPDF generates a printable PDF of upcoming rides at public/events.pdf.
func BuildPDF() error {
	return exec.Command("go", "run", "./cmd/buildpdf").Run()
}
