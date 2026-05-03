package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"gopkg.in/yaml.v3"
)

type frontMatter struct {
	Events []event `yaml:"events"`
}

type event struct {
	Title        string   `yaml:"title"`
	Date         string   `yaml:"date"`
	URL          string   `yaml:"url"`
	Start        string   `yaml:"start"`
	End          string   `yaml:"end"`
	StartAddress string   `yaml:"start_address"`
	Tags         []string `yaml:"tags"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	raw, err := os.ReadFile("content/events.md")
	if err != nil {
		return fmt.Errorf("reading events.md: %w", err)
	}

	parts := strings.SplitN(string(raw), "---", 3)
	if len(parts) < 3 {
		return fmt.Errorf("no YAML front matter in events.md")
	}

	var fm frontMatter
	if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
		return fmt.Errorf("parsing events YAML: %w", err)
	}

	today := time.Now().Truncate(24 * time.Hour)
	var upcoming []event
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

	sort.Slice(upcoming, func(i, j int) bool {
		ti, _ := time.Parse("January 2, 2006", upcoming[i].Date)
		tj, _ := time.Parse("January 2, 2006", upcoming[j].Date)
		return ti.Before(tj)
	})

	if len(upcoming) == 0 {
		fmt.Println("BuildPDF: no upcoming events found")
		return nil
	}

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

	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 5, fmt.Sprintf("%d upcoming rides as of %s", count, generated), "", 1, "L", false, 0, "")
	pdf.Ln(4)

	for _, e := range upcoming {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(20, 20, 20)
		pdf.MultiCell(0, 6, e.Title, "", "L", false)

		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(50, 50, 50)
		pdf.CellFormat(0, 5, e.Date, "", 1, "L", false, 0, "")

		if e.Start != "" {
			loc := e.Start
			if e.End != "" && e.End != e.Start {
				loc += " > " + e.End
			}
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(80, 80, 80)
			pdf.CellFormat(0, 4, "Location: "+loc, "", 1, "L", false, 0, "")
		}

		if len(e.Tags) > 0 {
			pdf.SetFont("Helvetica", "I", 8)
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(0, 4, strings.Join(e.Tags, ", "), "", 1, "L", false, 0, "")
		}

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
