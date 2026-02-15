//go:build mage

package main

import (
	"fmt"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

// Build builds TypeScript and Hugo site
func Build() error {
	mg.Deps(BuildTS)
	fmt.Println("Building Hugo site...")
	return sh.RunV("hugo", "--gc", "--minify")
}

// InstallNpmDeps installs npm dependencies
func InstallNpmDeps() error {
	fmt.Println("Installing npm dependencies...")
	return sh.RunV("npm", "install")
}

// BuildTS compiles TypeScript to JavaScript
func BuildTS() error {
	mg.Deps(InstallNpmDeps)
	fmt.Println("Compiling TypeScript...")

	// Ensure output directory exists
	outDir := "themes/linkpage/static/js"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return sh.RunV("esbuild",
		"src/main.ts",
		"--bundle",
		"--minify",
		"--sourcemap",
		"--target=es2020",
		"--outfile="+outDir+"/main.js",
	)
}

// Serve starts the Hugo development server (builds TS first)
func Serve() error {
	mg.Deps(BuildTS)
	return sh.RunV("hugo", "server", "-D")
}

// Dev runs TypeScript in watch mode alongside Hugo server
func Dev() error {
	mg.Deps(BuildTS)
	fmt.Println("Starting development server...")
	fmt.Println("Note: Run 'mage buildts' after TypeScript changes, or use 'mage watch' in another terminal")
	return sh.RunV("hugo", "server", "-D")
}

// Watch watches TypeScript files and rebuilds on change
func Watch() error {
	fmt.Println("Watching TypeScript files...")
	return sh.RunV("esbuild",
		"src/main.ts",
		"--bundle",
		"--sourcemap",
		"--target=es2020",
		"--outfile=themes/linkpage/static/js/main.js",
		"--watch",
	)
}

// Clean removes the public directory
func Clean() error {
	fmt.Println("Cleaning public directory...")
	return os.RemoveAll("public")
}
