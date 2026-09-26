package entity

// Additional bundled D2 export formats.
const (
	FormatPPTX  ExportFormat = "pptx"
	FormatGIF   ExportFormat = "gif"
	FormatASCII ExportFormat = "ascii"
)

// LayoutEngine identifies a bundled D2 layout engine.
type LayoutEngine string

const (
	LayoutDagre LayoutEngine = "dagre"
	LayoutELK   LayoutEngine = "elk"
	LayoutTALA  LayoutEngine = "tala"
)

// RenderOptions configures compilation, layout, rendering, and bounded board composition.
type RenderOptions struct {
	Layout    LayoutOptions
	Theme     ThemeOptions
	Font      FontOptions
	Animation AnimationOptions
	Boards    BoardOptions
	ASCII     ASCIIOptions
	Padding   *int64
	Center    *bool
	Scale     *float64
	Sketch    *bool
	Limits    RenderLimits
}

// LayoutOptions contains engine-specific public D2 v0.9 settings.
type LayoutOptions struct {
	Engine LayoutEngine
	Dagre  DagreLayoutOptions
	ELK    ELKLayoutOptions
	TALA   TALALayoutOptions
}

// DagreLayoutOptions configures Dagre spacing. Zero values use D2 defaults.
type DagreLayoutOptions struct {
	NodeSeparation int
	EdgeSeparation int
}

// ELKLayoutOptions configures public ELK layout settings. Zero values use D2 defaults.
type ELKLayoutOptions struct {
	Algorithm       string
	NodeSpacing     int
	Padding         string
	EdgeNodeSpacing int
	SelfLoopSpacing int
}

// TALALayoutOptions configures deterministic TALA attempts.
type TALALayoutOptions struct {
	Seeds          []int64
	MaxConcurrency int
}

// ThemeOptions configures light/dark themes and token overrides.
type ThemeOptions struct {
	LightID       *int64
	DarkID        *int64
	LightOverride map[string]string
	DarkOverride  map[string]string
}

// FontOptions selects bundled D2 font families.
type FontOptions struct {
	Regular string
	Mono    string
}

// AnimationOptions configures animated SVG/GIF sampling.
type AnimationOptions struct {
	IntervalMS int
	MaxFrames  int
}

// BoardOptions selects a board and controls descendant composition.
type BoardOptions struct {
	Path               []string
	IncludeDescendants *bool
	MaxBoards          int
}

// ASCIICharset selects the bundled ASCII renderer character set.
type ASCIICharset string

const (
	ASCIIExtended ASCIICharset = "extended"
	ASCIIStandard ASCIICharset = "standard"
)

// ASCIIOptions configures text rendering.
type ASCIIOptions struct {
	Charset ASCIICharset
}

// RenderLimits bounds raster and encoded output work.
type RenderLimits struct {
	MaxWidth       int
	MaxHeight      int
	MaxPixels      int64
	MaxOutputBytes int64
}
