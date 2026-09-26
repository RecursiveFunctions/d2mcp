package d2

import (
	"archive/zip"
	"bytes"
	"context"
	"image/gif"
	"io"
	"testing"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/domain/repository"
)

func renderWithOptions(t *testing.T, content string, format entity.ExportFormat, options entity.RenderOptions) []byte {
	t.Helper()
	repo := NewD2Repository()
	renderer, ok := repo.(repository.RenderOptionsRepository)
	if !ok {
		t.Fatal("D2 repository does not implement RenderOptionsRepository")
	}
	reader, err := renderer.RenderWithOptions(context.Background(), content, format, options)
	if err != nil {
		t.Fatalf("RenderWithOptions(%s) error = %v", format, err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read %s output: %v", format, err)
	}
	return output
}

func TestD2RepositoryBundledFormatsWithoutExternalCommands(t *testing.T) {
	t.Setenv("PATH", "")
	tests := []struct {
		format entity.ExportFormat
		magic  []byte
	}{
		{entity.FormatSVG, []byte("<?xml")},
		{entity.FormatPNG, []byte("\x89PNG\r\n\x1a\n")},
		{entity.FormatPDF, []byte("%PDF")},
		{entity.FormatPPTX, []byte("PK")},
		{entity.FormatGIF, []byte("GIF8")},
	}
	for _, test := range tests {
		t.Run(string(test.format), func(t *testing.T) {
			output := renderWithOptions(t, "a -> b: bundled", test.format, entity.RenderOptions{})
			if !bytes.HasPrefix(output, test.magic) {
				t.Fatalf("%s output prefix = %q, want %q", test.format, output[:min(len(output), len(test.magic))], test.magic)
			}
		})
	}
	ascii := renderWithOptions(t, "a -> b: bundled", entity.FormatASCII, entity.RenderOptions{
		ASCII: entity.ASCIIOptions{Charset: entity.ASCIIStandard},
	})
	if len(bytes.TrimSpace(ascii)) == 0 {
		t.Fatal("ASCII output is empty")
	}
}

func TestD2RepositoryBundledLayouts(t *testing.T) {
	for _, layout := range []entity.LayoutEngine{entity.LayoutDagre, entity.LayoutELK, entity.LayoutTALA} {
		t.Run(string(layout), func(t *testing.T) {
			options := entity.RenderOptions{Layout: entity.LayoutOptions{Engine: layout}}
			if layout == entity.LayoutTALA {
				options.Layout.TALA = entity.TALALayoutOptions{Seeds: []int64{7}, MaxConcurrency: 1}
			}
			output := renderWithOptions(t, "a -> b\nb -> c", entity.FormatSVG, options)
			if !bytes.Contains(output, []byte("<svg")) {
				t.Fatal("SVG output is missing svg element")
			}
		})
	}
}

func TestD2RepositoryTypedRenderOptions(t *testing.T) {
	padding := int64(24)
	center := true
	scale := 1.25
	sketch := false
	themeID := int64(1)
	output := renderWithOptions(t, "a -> b", entity.FormatSVG, entity.RenderOptions{
		Padding: &padding,
		Center:  &center,
		Scale:   &scale,
		Sketch:  &sketch,
		Theme: entity.ThemeOptions{
			LightID:       &themeID,
			LightOverride: map[string]string{"N1": "#ffffff"},
		},
		Font: entity.FontOptions{Regular: "source-sans", Mono: "source-code"},
	})
	if !bytes.Contains(output, []byte(`width="`)) {
		t.Fatal("scaled SVG is missing an explicit width")
	}
}

func TestD2RepositoryBoundedMultiBoardExports(t *testing.T) {
	content := "root: Root\nlayers: {\n  detail: {\n    a -> b\n  }\n}"
	for _, format := range []entity.ExportFormat{entity.FormatPDF, entity.FormatPPTX, entity.FormatGIF} {
		t.Run(string(format), func(t *testing.T) {
			output := renderWithOptions(t, content, format, entity.RenderOptions{})
			if len(output) == 0 {
				t.Fatalf("%s multi-board output is empty", format)
			}
			switch format {
			case entity.FormatPDF:
				if !bytes.Contains(output, []byte("/Count 2")) {
					t.Fatal("PDF does not declare two pages")
				}
			case entity.FormatPPTX:
				archive, err := zip.NewReader(bytes.NewReader(output), int64(len(output)))
				if err != nil {
					t.Fatalf("open PPTX: %v", err)
				}
				slides := 0
				for _, file := range archive.File {
					if file.Name == "ppt/slides/slide1.xml" || file.Name == "ppt/slides/slide2.xml" {
						slides++
					}
				}
				if slides != 2 {
					t.Fatalf("PPTX slide count = %d, want 2", slides)
				}
			case entity.FormatGIF:
				decoded, err := gif.DecodeAll(bytes.NewReader(output))
				if err != nil {
					t.Fatalf("decode GIF: %v", err)
				}
				if len(decoded.Image) != 2 {
					t.Fatalf("GIF frame count = %d, want 2", len(decoded.Image))
				}
			}
		})
	}

	repo := NewD2Repository().(repository.RenderOptionsRepository)
	_, err := repo.RenderWithOptions(context.Background(), content, entity.FormatPDF, entity.RenderOptions{
		Boards: entity.BoardOptions{MaxBoards: 1},
	})
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("board count exceeds")) {
		t.Fatalf("board limit error = %v", err)
	}

	include := true
	animated := renderWithOptions(t, content, entity.FormatSVG, entity.RenderOptions{
		Boards:    entity.BoardOptions{IncludeDescendants: &include},
		Animation: entity.AnimationOptions{IntervalMS: 250},
	})
	if !bytes.Contains(animated, []byte("@keyframes")) {
		t.Fatal("multi-board SVG is not animated")
	}
}
