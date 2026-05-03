//go:build mage

package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"gopkg.in/yaml.v3"
)

type pdfFrontMatter struct {
	Events []pdfEvent `yaml:"events"`
}

type pdfEvent struct {
	Title        string   `yaml:"title"`
	Date         string   `yaml:"date"`
	URL          string   `yaml:"url"`
	Start        string   `yaml:"start"`
	End          string   `yaml:"end"`
	StartAddress string   `yaml:"start_address"`
	Tags         []string `yaml:"tags"`
}

// BuildPDF generates a printable PDF of upcoming rides at public/events.pdf.
func BuildPDF() error {
	raw, err := os.ReadFile("content/events.md")
	if err != nil {
		return fmt.Errorf("reading events.md: %w", err)
	}

	// Extract YAML front matter between --- delimiters
	parts := strings.SplitN(string(raw), "---", 3)
	if len(parts) < 3 {
		return fmt.Errorf("no YAML front matter in events.md")
	}

	var fm pdfFrontMatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return fmt.Errorf("parsing events YAML: %w", err)
	}

	// Filter to upcoming events (today and future)
	today := time.Now().Truncate(24 * time.Hour)
	var upcoming []pdfEvent
	for _, e := range fm.Events {
		if e.Date == "" {
			continue
		}
		t, err := time.Parse("January 2, 2006", e.Date)
		if err != nil {
			continue
		}
		if !t.Before(today) {
			upcoming = append(upcoming, e)
		}
	}

	if len(upcoming) == 0 {
		fmt.Println("BuildPDF: no upcoming events found")
		return nil
	}

	// Generate QR code pointing to the site
	qrPNG, err := qrcode.Encode("https://ridewestside.org", qrcode.Medium, 128)
	if err != nil {
		return fmt.Errorf("generating QR code: %w", err)
	}

	generated := time.Now().Format("January 2, 2006")
	count := len(upcoming)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 30, 20)
	pdf.SetAutoPageBreak(true, 20)
	pdf.RegisterImageOptionsReader("qr", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(qrPNG))

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Helvetica", "B", 13)
		pdf.SetTextColor(30, 30, 30)
		pdf.SetY(8)
		pdf.CellFormat(160, 8, "Ride Westside - Upcoming Rides", "", 0, "L", false, 0, "")
		pdf.Image("qr", 172, 6, 16, 16, false, "", 0, "https://ridewestside.org")
		pdf.SetDrawColor(74, 222, 128)
		pdf.SetLineWidth(0.4)
		pdf.Line(20, 27, 190, 27)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 6,
			fmt.Sprintf("Page %d  |  ridewestside.org  |  Generated %s", pdf.PageNo(), generated),
			"", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	// Summary line
	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 5, fmt.Sprintf("%d upcoming rides as of %s", count, generated), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	for _, e := range upcoming {
		// Title
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(20, 20, 20)
		pdf.MultiCell(0, 6, e.Title, "", "L", false)

		// Date
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(50, 50, 50)
		pdf.CellFormat(0, 5, e.Date, "", 1, "L", false, 0, "")

		// Location
		if e.Start != "" {
			loc := e.Start
			if e.End != "" && e.End != e.Start {
				loc += " > " + e.End
			}
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(0, 4, "Location: "+loc, "", 1, "L", false, 0, "")
		}

		// Tags
		if len(e.Tags) > 0 {
			pdf.SetFont("Helvetica", "I", 8)
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(0, 4, strings.Join(e.Tags, ", "), "", 1, "L", false, 0, "")
		}

		// URL as clickable link
		if e.URL != "" {
			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(0, 80, 180)
			pdf.CellFormat(0, 4, e.URL, "", 1, "L", false, 0, e.URL)
		}

		pdf.Ln(5)
	}

	if err := os.MkdirAll("public", 0755); err != nil {
		return err
	}
	outPath := "public/events.pdf"
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}
	fmt.Printf("BuildPDF: wrote %d events to %s\n", count, outPath)
	return nil
}
