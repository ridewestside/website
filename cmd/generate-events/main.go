// Command generate-events expands content/events.md's `recurring` rules
// into data/generated_events.yaml. It's the CI equivalent of `mage
// generateRecurringEvents`, for pipelines (GitHub Actions, Netlify) that
// build the site without mage installed — see cmd/buildpdf for the same
// pattern applied to the PDF build.
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ridewestside/website/internal/recurring"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	n, err := recurring.GenerateDataFile("content/events.md", "data/generated_events.yaml", time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("Generated %d recurring event occurrence(s) into data/generated_events.yaml\n", n)
	return nil
}
