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

const (
	pageW      = 210.0
	pageH      = 297.0
	marginL    = 20.0
	marginR    = 20.0
	marginTop  = 48.0 // enough room for a 24mm QR starting at y=6, divider at y=34, plus clearance
	marginBot  = 22.0
	contentW   = pageW - marginL - marginR // 170mm
	evtQRSz    = 22.0                      // per-event QR code size
	evtQRGap   = 4.0
	textColW   = contentW - evtQRSz - evtQRGap // 144mm
	evtQRX     = marginL + textColW + evtQRGap  // x=168mm
	headerQRSz = 24.0
	headerQRX  = pageW - marginR - headerQRSz // x=166mm
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

	sort.SliceStable(upcoming, func(i, j int) bool {
		ti, _ := time.Parse("January 2, 2006", upcoming[i].Date)
		tj, _ := time.Parse("January 2, 2006", upcoming[j].Date)
		if ti.Equal(tj) {
			return upcoming[i].Title < upcoming[j].Title
		}
		return ti.Before(tj)
	})

	if len(upcoming) == 0 {
		fmt.Println("BuildPDF: no upcoming events found")
		return nil
	}

	siteQRPNG, err := qrcode.Encode("https://ridewestside.org", qrcode.Medium, 256)
	if err != nil {
		return fmt.Errorf("generating site QR: %w", err)
	}

	// Pre-generate a QR PNG for each unique event URL.
	urlQRPNG := make(map[string][]byte)
	for _, e := range upcoming {
		if e.URL == "" {
			continue
		}
		if _, seen := urlQRPNG[e.URL]; seen {
			continue
		}
		png, err := qrcode.Encode(e.URL, qrcode.Medium, 192)
		if err == nil {
			urlQRPNG[e.URL] = png
		}
	}

	generated := time.Now().Format("January 2, 2006")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginL, marginTop, marginR)
	pdf.SetAutoPageBreak(true, marginBot)

	pdf.RegisterImageOptionsReader("site-qr",
		gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(siteQRPNG))

	// Register each unique URL QR under its URL as the key.
	for url, png := range urlQRPNG {
		pdf.RegisterImageOptionsReader(url,
			gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(png))
	}

	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetTextColor(30, 30, 30)
		pdf.SetXY(marginL, 8)
		pdf.CellFormat(headerQRX-marginL-4, headerQRSz, "Ride Westside", "", 0, "LM", false, 0, "")
		pdf.Image("site-qr", headerQRX, 6, headerQRSz, headerQRSz, false, "", 0, "https://ridewestside.org")
		pdf.SetDrawColor(74, 222, 128)
		pdf.SetLineWidth(0.5)
		// Divider sits 8mm below the QR bottom, well above marginTop.
		pdf.Line(marginL, 6+headerQRSz+6, pageW-marginR, 6+headerQRSz+6)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(140, 140, 140)
		pdf.CellFormat(0, 6,
			fmt.Sprintf("Page %d  |  ridewestside.org  |  Generated %s", pdf.PageNo(), generated),
			"", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	for _, e := range upcoming {
		hasQR := e.URL != "" && urlQRPNG[e.URL] != nil
		colW := contentW
		if hasQR {
			colW = textColW
		}

		// Estimate event block height so we can page-break before splitting an event.
		estH := 6.0 + 5.0 // title + date
		if e.Start != "" {
			estH += 4.5
		}
		if len(e.Tags) > 0 {
			estH += 4.5
		}
		if e.URL != "" {
			estH += 4.5
		}
		if hasQR && evtQRSz > estH {
			estH = evtQRSz
		}
		estH += 8.0 // bottom gap + divider

		if pdf.GetY()+estH > pageH-marginBot {
			pdf.AddPage()
		}

		startY := pdf.GetY()

		// Title
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(20, 20, 20)
		pdf.SetX(marginL)
		pdf.MultiCell(colW, 6, e.Title, "", "L", false)

		// Date
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(60, 60, 60)
		pdf.SetX(marginL)
		pdf.CellFormat(colW, 5, e.Date, "", 1, "L", false, 0, "")

		// Location
		if e.Start != "" {
			loc := e.Start
			if e.End != "" && e.End != e.Start {
				loc += " → " + e.End
			}
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetTextColor(90, 90, 90)
			pdf.SetX(marginL)
			pdf.CellFormat(colW, 4.5, loc, "", 1, "L", false, 0, "")
		}

		// Tags
		if len(e.Tags) > 0 {
			pdf.SetFont("Helvetica", "I", 8)
			pdf.SetTextColor(110, 110, 110)
			pdf.SetX(marginL)
			pdf.CellFormat(colW, 4.5, strings.Join(e.Tags, "  ·  "), "", 1, "L", false, 0, "")
		}

		// URL
		if e.URL != "" {
			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(0, 90, 200)
			pdf.SetX(marginL)
			pdf.CellFormat(colW, 4.5, e.URL, "", 1, "L", false, 0, e.URL)
		}

		textEndY := pdf.GetY()

		// Per-event QR code, vertically centered relative to the text block.
		if hasQR {
			qrY := startY + (textEndY-startY-evtQRSz)/2.0
			if qrY < startY {
				qrY = startY
			}
			pdf.Image(e.URL, evtQRX, qrY, evtQRSz, evtQRSz, false, "", 0, e.URL)
			if qrY+evtQRSz > textEndY {
				textEndY = qrY + evtQRSz
			}
		}

		// Divider between events.
		endY := textEndY + 4.0
		pdf.SetDrawColor(220, 220, 220)
		pdf.SetLineWidth(0.2)
		pdf.Line(marginL, endY, pageW-marginR, endY)
		pdf.SetY(endY + 4.0)
	}

	if err := os.MkdirAll("public", 0755); err != nil {
		return err
	}
	outPath := "public/events.pdf"
	if err := pdf.OutputFileAndClose(outPath); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}
	fmt.Printf("BuildPDF: wrote %d events to %s\n", len(upcoming), outPath)
	return nil
}
