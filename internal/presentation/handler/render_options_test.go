package handler

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/domain/entity"
)

func TestParseRenderOptions(t *testing.T) {
	request := mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: map[string]any{
		"layout":              "tala",
		"talaSeeds":           []any{float64(3), float64(5)},
		"talaMaxConcurrency":  float64(2),
		"themeId":             float64(1),
		"themeOverrides":      map[string]any{"N1": "#ffffff"},
		"fontFamily":          "source-sans",
		"monoFontFamily":      "source-code",
		"padding":             float64(20),
		"center":              true,
		"scale":               1.5,
		"sketch":              false,
		"boardPath":           "layers.detail",
		"includeDescendants":  false,
		"animationIntervalMs": float64(250),
		"asciiCharset":        "standard",
	}}}
	options, err := parseRenderOptions(request)
	if err != nil {
		t.Fatalf("parseRenderOptions() error = %v", err)
	}
	if options.Layout.Engine != entity.LayoutTALA || len(options.Layout.TALA.Seeds) != 2 || options.Layout.TALA.MaxConcurrency != 2 {
		t.Fatalf("TALA options = %#v", options.Layout)
	}
	if options.Theme.LightID == nil || *options.Theme.LightID != 1 || options.Theme.LightOverride["N1"] != "#ffffff" {
		t.Fatalf("theme options = %#v", options.Theme)
	}
	if len(options.Boards.Path) != 2 || options.Boards.Path[0] != "layers" || options.Boards.Path[1] != "detail" {
		t.Fatalf("board path = %#v", options.Boards.Path)
	}
	if options.Animation.IntervalMS != 250 || options.ASCII.Charset != entity.ASCIIStandard {
		t.Fatalf("animation/ascii options = %#v / %#v", options.Animation, options.ASCII)
	}
}

func TestExportMIMETypes(t *testing.T) {
	tests := map[entity.ExportFormat]string{
		entity.FormatSVG:   "image/svg+xml",
		entity.FormatPNG:   "image/png",
		entity.FormatPDF:   "application/pdf",
		entity.FormatPPTX:  "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		entity.FormatGIF:   "image/gif",
		entity.FormatASCII: "text/plain; charset=utf-8",
	}
	for format, want := range tests {
		if got := getMimeType(format); got != want {
			t.Errorf("getMimeType(%q) = %q, want %q", format, got, want)
		}
	}
}
