package handler

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/i2y/d2mcp/internal/domain/entity"
)

type renderArguments struct {
	Layout             string            `json:"layout"`
	DagreNodeSep       int               `json:"dagreNodeSep"`
	DagreEdgeSep       int               `json:"dagreEdgeSep"`
	ELKAlgorithm       string            `json:"elkAlgorithm"`
	ELKNodeSpacing     int               `json:"elkNodeSpacing"`
	ELKPadding         string            `json:"elkPadding"`
	ELKEdgeNodeSpacing int               `json:"elkEdgeNodeSpacing"`
	ELKSelfLoopSpacing int               `json:"elkSelfLoopSpacing"`
	TALASeeds          []int64           `json:"talaSeeds"`
	TALAMaxConcurrency int               `json:"talaMaxConcurrency"`
	ThemeID            *int64            `json:"themeId"`
	DarkThemeID        *int64            `json:"darkThemeId"`
	ThemeOverrides     map[string]string `json:"themeOverrides"`
	DarkThemeOverrides map[string]string `json:"darkThemeOverrides"`
	FontFamily         string            `json:"fontFamily"`
	MonoFontFamily     string            `json:"monoFontFamily"`
	Padding            *int64            `json:"padding"`
	Center             *bool             `json:"center"`
	Scale              *float64          `json:"scale"`
	Sketch             *bool             `json:"sketch"`
	BoardPath          string            `json:"boardPath"`
	IncludeDescendants *bool             `json:"includeDescendants"`
	MaxBoards          int               `json:"maxBoards"`
	AnimationInterval  int               `json:"animationIntervalMs"`
	MaxFrames          int               `json:"maxFrames"`
	ASCIICharset       string            `json:"asciiCharset"`
	MaxWidth           int               `json:"maxWidth"`
	MaxHeight          int               `json:"maxHeight"`
	MaxPixels          int64             `json:"maxPixels"`
	MaxOutputBytes     int64             `json:"maxOutputBytes"`
}

func parseRenderOptions(request mcp.CallToolRequest) (entity.RenderOptions, error) {
	var arguments renderArguments
	if err := request.BindArguments(&arguments); err != nil {
		return entity.RenderOptions{}, fmt.Errorf("invalid render options: %w", err)
	}
	var boardPath []string
	if arguments.BoardPath != "" {
		for _, segment := range strings.Split(arguments.BoardPath, ".") {
			if segment == "" {
				return entity.RenderOptions{}, fmt.Errorf("boardPath contains an empty segment")
			}
			boardPath = append(boardPath, segment)
		}
	}
	return entity.RenderOptions{
		Layout: entity.LayoutOptions{
			Engine: entity.LayoutEngine(arguments.Layout),
			Dagre:  entity.DagreLayoutOptions{NodeSeparation: arguments.DagreNodeSep, EdgeSeparation: arguments.DagreEdgeSep},
			ELK: entity.ELKLayoutOptions{
				Algorithm: arguments.ELKAlgorithm, NodeSpacing: arguments.ELKNodeSpacing, Padding: arguments.ELKPadding,
				EdgeNodeSpacing: arguments.ELKEdgeNodeSpacing, SelfLoopSpacing: arguments.ELKSelfLoopSpacing,
			},
			TALA: entity.TALALayoutOptions{Seeds: arguments.TALASeeds, MaxConcurrency: arguments.TALAMaxConcurrency},
		},
		Theme: entity.ThemeOptions{
			LightID: arguments.ThemeID, DarkID: arguments.DarkThemeID,
			LightOverride: arguments.ThemeOverrides, DarkOverride: arguments.DarkThemeOverrides,
		},
		Font:      entity.FontOptions{Regular: arguments.FontFamily, Mono: arguments.MonoFontFamily},
		Animation: entity.AnimationOptions{IntervalMS: arguments.AnimationInterval, MaxFrames: arguments.MaxFrames},
		Boards:    entity.BoardOptions{Path: boardPath, IncludeDescendants: arguments.IncludeDescendants, MaxBoards: arguments.MaxBoards},
		ASCII:     entity.ASCIIOptions{Charset: entity.ASCIICharset(arguments.ASCIICharset)},
		Padding:   arguments.Padding, Center: arguments.Center, Scale: arguments.Scale, Sketch: arguments.Sketch,
		Limits: entity.RenderLimits{
			MaxWidth: arguments.MaxWidth, MaxHeight: arguments.MaxHeight,
			MaxPixels: arguments.MaxPixels, MaxOutputBytes: arguments.MaxOutputBytes,
		},
	}, nil
}

func renderToolOptions() []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithString("layout", mcp.Description("Bundled layout engine"), mcp.Enum("dagre", "elk", "tala")),
		mcp.WithNumber("dagreNodeSep", mcp.Description("Dagre node separation"), mcp.Min(1)),
		mcp.WithNumber("dagreEdgeSep", mcp.Description("Dagre edge separation"), mcp.Min(1)),
		mcp.WithString("elkAlgorithm", mcp.Description("ELK algorithm name")),
		mcp.WithNumber("elkNodeSpacing", mcp.Description("ELK node spacing"), mcp.Min(1)),
		mcp.WithString("elkPadding", mcp.Description("ELK padding expression")),
		mcp.WithNumber("elkEdgeNodeSpacing", mcp.Description("ELK edge-to-node spacing"), mcp.Min(1)),
		mcp.WithNumber("elkSelfLoopSpacing", mcp.Description("ELK self-loop spacing"), mcp.Min(1)),
		mcp.WithArray("talaSeeds", mcp.Description("Deterministic signed 64-bit TALA seeds"), mcp.Items(map[string]any{"type": "integer"}), mcp.MaxItems(16)),
		mcp.WithNumber("talaMaxConcurrency", mcp.Description("Maximum concurrent TALA seed attempts"), mcp.Min(1), mcp.Max(16)),
		mcp.WithNumber("themeId", mcp.Description("Light theme ID")),
		mcp.WithNumber("darkThemeId", mcp.Description("Dark theme ID")),
		mcp.WithObject("themeOverrides", mcp.Description("Light theme token overrides (N1-N7, B1-B6, AA2/4/5, AB4/5)"), mcp.AdditionalProperties(map[string]any{"type": "string"})),
		mcp.WithObject("darkThemeOverrides", mcp.Description("Dark theme token overrides"), mcp.AdditionalProperties(map[string]any{"type": "string"})),
		mcp.WithString("fontFamily", mcp.Description("Bundled regular font family"), mcp.Enum("source-sans", "source-code", "hand-drawn")),
		mcp.WithString("monoFontFamily", mcp.Description("Bundled mono font family"), mcp.Enum("source-sans", "source-code", "hand-drawn")),
		mcp.WithNumber("padding", mcp.Description("Diagram padding"), mcp.Min(0), mcp.Max(4096)),
		mcp.WithBoolean("center", mcp.Description("Center diagram content")),
		mcp.WithNumber("scale", mcp.Description("Output scale"), mcp.Min(0.000001), mcp.Max(16)),
		mcp.WithBoolean("sketch", mcp.Description("Enable sketch rendering")),
		mcp.WithString("boardPath", mcp.Description("Dot-separated board path, for example layers.detail")),
		mcp.WithBoolean("includeDescendants", mcp.Description("Compose descendant boards where the format supports it")),
		mcp.WithNumber("maxBoards", mcp.Description("Per-operation board limit"), mcp.Min(1), mcp.Max(256)),
		mcp.WithNumber("animationIntervalMs", mcp.Description("Board interval and style-animation sampling duration"), mcp.Min(0), mcp.Max(60000)),
		mcp.WithNumber("maxFrames", mcp.Description("Per-operation GIF frame limit"), mcp.Min(1), mcp.Max(600)),
		mcp.WithString("asciiCharset", mcp.Description("ASCII character set"), mcp.Enum("extended", "standard")),
		mcp.WithNumber("maxWidth", mcp.Description("Maximum raster width"), mcp.Min(1), mcp.Max(16384)),
		mcp.WithNumber("maxHeight", mcp.Description("Maximum raster height"), mcp.Min(1), mcp.Max(16384)),
		mcp.WithNumber("maxPixels", mcp.Description("Maximum pixels per raster frame"), mcp.Min(1), mcp.Max(67108864)),
		mcp.WithNumber("maxOutputBytes", mcp.Description("Maximum encoded output bytes"), mcp.Min(1), mcp.Max(268435456)),
	}
}
