// ogimage – generates Open Graph social preview images for blog posts.
//
// Reads Markdown front matter from content/post/*.md, renders a styled card
// with the post title, and saves PNG images to static/assets/og/<slug>.png.
//
// Usage:
//
//	go run ./cmd/ogimage
//	go run ./cmd/ogimage --glob="src/content/post/*.md" --out="src/static/assets/og"
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const (
	imgW = 1200
	imgH = 630
)

func main() {
	glob := flag.String("glob", "content/post/*.md", "Glob pattern for markdown files")
	outDir := flag.String("out", "static/assets/og", "Output directory for PNG images")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	files, err := filepath.Glob(*glob)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Glob error: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found.")
		return
	}

	// Load TrueType font
	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse font: %v\n", err)
		os.Exit(1)
	}
	titleFace, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    42,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create title face: %v\n", err)
		os.Exit(1)
	}
	authorFace, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create author face: %v\n", err)
		os.Exit(1)
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", f, err)
			continue
		}

		fm := parseFrontMatter(string(content))
		title := fm["title"]
		if title == "" {
			title = humanizeSlug(strings.TrimSuffix(filepath.Base(f), ".md"))
		}

		slug := strings.TrimSuffix(filepath.Base(f), ".md")
		outPath := filepath.Join(*outDir, slug+".png")

		img := generateCard(title, fm["author"], titleFace, authorFace)
		if err := savePNG(img, outPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save %s: %v\n", outPath, err)
			continue
		}

		fmt.Printf("Generated %s → %s\n", slug, outPath)
	}
}

func parseFrontMatter(md string) map[string]string {
	re := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	m := re.FindStringSubmatch(md)
	if m == nil {
		return map[string]string{}
	}
	fm := make(map[string]string)
	for _, line := range strings.Split(m[1], "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := regexp.MustCompile(`^(\w[\w-]*):\s*(.*)$`).FindStringSubmatch(line)
		if kv == nil {
			continue
		}
		val := strings.TrimSpace(kv[2])
		val = strings.Trim(val, `"'`)
		fm[kv[1]] = val
	}
	return fm
}

func humanizeSlug(s string) string {
	s = strings.ReplaceAll(s, "-", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func generateCard(title, author string, titleFace, authorFace font.Face) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))

	// Background gradient
	bgStart := color.RGBA{210, 233, 248, 255}
	bgEnd := color.RGBA{180, 215, 240, 255}
	for y := 0; y < imgH; y++ {
		t := float64(y) / float64(imgH)
		c := lerp(bgStart, bgEnd, t)
		for x := 0; x < imgW; x++ {
			img.Set(x, y, c)
		}
	}

	// Top accent bar
	accent := color.RGBA{99, 179, 237, 255}
	for y := 0; y < 6; y++ {
		for x := 0; x < imgW; x++ {
			img.Set(x, y, accent)
		}
	}

	// Decorative circle
	drawCircle(img, imgW-200, 160, 280, color.RGBA{180, 215, 240, 50})

	// Title (wrapped and centered)
	titleLines := wrapLines(titleFace, title, imgW-200)
	y := 200
	for _, line := range titleLines {
		drawString(img, line, titleFace, y, color.RGBA{15, 23, 42, 255})
		y += 56
	}

	// Author
	if author != "" {
		drawString(img, "by "+author, authorFace, imgH-100, color.RGBA{71, 85, 105, 255})
	}

	// Bottom accent
	for y := imgH - 6; y < imgH; y++ {
		for x := 0; x < imgW; x++ {
			img.Set(x, y, accent)
		}
	}

	return img
}

func wrapLines(face font.Face, text string, maxWidth int) []string {
	var lines []string
	words := strings.Fields(text)
	current := ""

	for _, word := range words {
		test := current
		if test != "" {
			test += " "
		}
		test += word
		w := textWidth(face, test)
		if w > maxWidth && current != "" {
			lines = append(lines, current)
			current = word
		} else {
			current = test
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func textWidth(face font.Face, text string) int {
	d := font.Drawer{Face: face}
	return d.MeasureString(text).Round()
}

func drawString(img *image.RGBA, text string, face font.Face, y int, c color.Color) {
	w := textWidth(face, text)
	x := (imgW - w) / 2

	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(text)
}

func drawCircle(img *image.RGBA, cx, cy, r int, c color.Color) {
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				img.Set(cx+dx, cy+dy, c)
			}
		}
	}
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

func savePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
