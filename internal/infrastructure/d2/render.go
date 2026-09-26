package d2

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/d2lang/d2/d2graph"
	"github.com/d2lang/d2/d2layouts/d2dagrelayout"
	"github.com/d2lang/d2/d2layouts/d2elklayout"
	"github.com/d2lang/d2/d2layouts/d2talalayout"
	"github.com/d2lang/d2/d2lib"
	"github.com/d2lang/d2/d2renderers/d2animate"
	"github.com/d2lang/d2/d2renderers/d2ascii"
	"github.com/d2lang/d2/d2renderers/d2ascii/charset"
	"github.com/d2lang/d2/d2renderers/d2fonts"
	"github.com/d2lang/d2/d2renderers/d2raster"
	"github.com/d2lang/d2/d2renderers/d2scene"
	"github.com/d2lang/d2/d2renderers/d2scenebuild"
	"github.com/d2lang/d2/d2renderers/d2svg"
	"github.com/d2lang/d2/d2renderers/d2svgimport"
	"github.com/d2lang/d2/d2target"
	"github.com/d2lang/d2/lib/imageasset"
	"github.com/d2lang/d2/lib/pdf"
	"github.com/d2lang/d2/lib/pptx"
	"github.com/d2lang/d2/lib/textmeasure"
	"github.com/d2lang/d2/lib/xgif"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/security/workspace"
)

const (
	defaultMaxBoards      = 64
	hardMaxBoards         = 256
	defaultMaxDimension   = 8192
	hardMaxDimension      = 16384
	defaultMaxPixels      = int64(32 * 1024 * 1024)
	hardMaxPixels         = int64(64 * 1024 * 1024)
	defaultMaxOutputBytes = int64(64 * 1024 * 1024)
	hardMaxOutputBytes    = int64(256 * 1024 * 1024)
	defaultMaxFrames      = 120
	hardMaxFrames         = 600
	maxAggregatePixels    = int64(128 * 1024 * 1024)
)

type normalizedRenderOptions struct {
	entity.RenderOptions
	maxBoards      int
	maxWidth       int
	maxHeight      int
	maxPixels      int64
	maxOutputBytes int64
	maxFrames      int
}

type renderedBoard struct {
	diagram *d2target.Diagram
	id      string
	names   []string
}

// RenderWithOptions renders content entirely through public D2 v0.9 Go APIs.
func (r *D2Repository) RenderWithOptions(ctx context.Context, content string, format entity.ExportFormat, options entity.RenderOptions) (io.Reader, error) {
	return r.renderWithOptionsInRoot(ctx, workspace.DefaultRoot, content, format, options)
}

func (r *D2Repository) renderWithOptionsInRoot(ctx context.Context, rootName, content string, format entity.ExportFormat, options entity.RenderOptions) (io.Reader, error) {
	var output []byte
	err := withSilentD2(ctx, func(ctx context.Context) error {
		normalized, err := normalizeRenderOptions(options)
		if err != nil {
			return err
		}
		format = normalizeFormat(format)
		if !supportedFormat(format) {
			return fmt.Errorf("unsupported format %q; expected svg, png, pdf, pptx, gif, or ascii", format)
		}
		diagram, renderOpts, err := r.compileForRender(ctx, rootName, content, normalized)
		if err != nil {
			return err
		}
		if err := r.secureDiagramAssets(ctx, rootName, diagram); err != nil {
			return err
		}
		imageResolver, err := r.newImageAssetResolver(rootName)
		if err != nil {
			return fmt.Errorf("configure image assets: %w", err)
		}
		selected, err := selectBoard(diagram, normalized.Boards.Path)
		if err != nil {
			return err
		}
		output, err = renderCompiled(ctx, selected, format, renderOpts, normalized, imageResolver)
		if err != nil {
			return err
		}
		if int64(len(output)) > normalized.maxOutputBytes {
			return fmt.Errorf("rendered output is %d bytes, exceeding the %d-byte limit", len(output), normalized.maxOutputBytes)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(output), nil
}

func normalizeFormat(format entity.ExportFormat) entity.ExportFormat {
	if format == "" {
		return entity.FormatSVG
	}
	if format == "txt" {
		return entity.FormatASCII
	}
	return format
}

func supportedFormat(format entity.ExportFormat) bool {
	switch format {
	case entity.FormatSVG, entity.FormatPNG, entity.FormatPDF, entity.FormatPPTX, entity.FormatGIF, entity.FormatASCII:
		return true
	default:
		return false
	}
}

func normalizeRenderOptions(options entity.RenderOptions) (normalizedRenderOptions, error) {
	n := normalizedRenderOptions{RenderOptions: options}
	var err error
	if n.maxBoards, err = boundedInt("max boards", options.Boards.MaxBoards, defaultMaxBoards, hardMaxBoards); err != nil {
		return n, err
	}
	if n.maxWidth, err = boundedInt("max width", options.Limits.MaxWidth, defaultMaxDimension, hardMaxDimension); err != nil {
		return n, err
	}
	if n.maxHeight, err = boundedInt("max height", options.Limits.MaxHeight, defaultMaxDimension, hardMaxDimension); err != nil {
		return n, err
	}
	if n.maxPixels, err = boundedInt64("max pixels", options.Limits.MaxPixels, defaultMaxPixels, hardMaxPixels); err != nil {
		return n, err
	}
	if n.maxOutputBytes, err = boundedInt64("max output bytes", options.Limits.MaxOutputBytes, defaultMaxOutputBytes, hardMaxOutputBytes); err != nil {
		return n, err
	}
	if n.maxFrames, err = boundedInt("max frames", options.Animation.MaxFrames, defaultMaxFrames, hardMaxFrames); err != nil {
		return n, err
	}
	if options.Animation.IntervalMS < 0 || options.Animation.IntervalMS > 60_000 {
		return n, fmt.Errorf("animation interval must be between 0 and 60000 milliseconds")
	}
	if options.Scale != nil && (*options.Scale <= 0 || *options.Scale > 16) {
		return n, fmt.Errorf("scale must be greater than zero and at most 16")
	}
	if options.Padding != nil && (*options.Padding < 0 || *options.Padding > 4096) {
		return n, fmt.Errorf("padding must be between 0 and 4096")
	}
	return n, nil
}

func boundedInt(name string, value, fallback, maximum int) (int, error) {
	if value == 0 {
		return fallback, nil
	}
	if value < 0 || value > maximum {
		return 0, fmt.Errorf("%s must be between 1 and %d", name, maximum)
	}
	return value, nil
}

func boundedInt64(name string, value, fallback, maximum int64) (int64, error) {
	if value == 0 {
		return fallback, nil
	}
	if value < 0 || value > maximum {
		return 0, fmt.Errorf("%s must be between 1 and %d", name, maximum)
	}
	return value, nil
}

func (r *D2Repository) compileForRender(ctx context.Context, rootName, content string, options normalizedRenderOptions) (*d2target.Diagram, d2svg.RenderOpts, error) {
	workspaces := r.workspaces
	if workspaces == nil {
		var err error
		workspaces, err = workspace.New("", nil)
		if err != nil {
			return nil, d2svg.RenderOpts{}, err
		}
	}
	filesystem, err := workspaces.FS(rootName)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("select workspace root: %w", err)
	}
	ruler, err := textmeasure.NewRuler()
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("create text ruler: %w", err)
	}
	regular, err := bundledFont(options.Font.Regular)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("regular font: %w", err)
	}
	mono, err := bundledFont(options.Font.Mono)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("mono font: %w", err)
	}
	layout, resolver, err := layoutResolver(options.Layout)
	if err != nil {
		return nil, d2svg.RenderOpts{}, err
	}
	lightOverrides, err := themeOverrides(options.Theme.LightOverride)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("light theme override: %w", err)
	}
	darkOverrides, err := themeOverrides(options.Theme.DarkOverride)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("dark theme override: %w", err)
	}
	renderOpts := d2svg.RenderOpts{
		Pad:                options.Padding,
		Center:             options.Center,
		Scale:              options.Scale,
		Sketch:             options.Sketch,
		ThemeID:            options.Theme.LightID,
		DarkThemeID:        options.Theme.DarkID,
		ThemeOverrides:     lightOverrides,
		DarkThemeOverrides: darkOverrides,
	}
	compileOpts := &d2lib.CompileOptions{
		Ruler:          ruler,
		Layout:         layout,
		LayoutResolver: resolver,
		FontFamily:     regular,
		MonoFontFamily: mono,
		FS:             filesystem,
		InputPath:      "index.d2",
	}
	diagram, _, err := d2lib.Compile(ctx, content, compileOpts, &renderOpts)
	if err != nil {
		return nil, d2svg.RenderOpts{}, fmt.Errorf("compile D2: %w", err)
	}
	return diagram, renderOpts, nil
}

func (r *D2Repository) secureDiagramAssets(ctx context.Context, rootName string, diagram *d2target.Diagram) error {
	if r.assetResolver == nil {
		return nil
	}
	resolved := make(map[string]*url.URL)
	var secureURL = func(source **url.URL) error {
		if source == nil || *source == nil {
			return nil
		}
		reference := (*source).String()
		if cached := resolved[reference]; cached != nil {
			*source = cached
			return nil
		}
		dataURI, err := r.assetResolver.DataURI(ctx, rootName, reference)
		if err != nil {
			return fmt.Errorf("resolve image asset %q: %w", reference, err)
		}
		parsed, err := url.Parse(dataURI)
		if err != nil {
			return fmt.Errorf("parse resolved image asset: %w", err)
		}
		resolved[reference] = parsed
		*source = parsed
		return nil
	}

	var walk func(*d2target.Diagram) error
	walk = func(board *d2target.Diagram) error {
		if board == nil {
			return nil
		}
		for i := range board.Shapes {
			if err := secureURL(&board.Shapes[i].Icon); err != nil {
				return err
			}
		}
		for i := range board.Connections {
			if err := secureURL(&board.Connections[i].Icon); err != nil {
				return err
			}
		}
		if board.Legend != nil {
			for i := range board.Legend.Shapes {
				if err := secureURL(&board.Legend.Shapes[i].Icon); err != nil {
					return err
				}
			}
			for i := range board.Legend.Connections {
				if err := secureURL(&board.Legend.Connections[i].Icon); err != nil {
					return err
				}
			}
		}
		for _, children := range [][]*d2target.Diagram{board.Layers, board.Scenarios, board.Steps} {
			for _, child := range children {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(diagram)
}

func (r *D2Repository) newImageAssetResolver(rootName string) (*imageasset.Resolver, error) {
	if r.workspaces == nil || r.httpClient == nil {
		return nil, nil
	}
	root, err := r.workspaces.Root(rootName)
	if err != nil {
		return nil, err
	}
	maxAssetBytes := r.maxAssetBytes
	if maxAssetBytes == 0 {
		maxAssetBytes = 10 << 20
	}
	return imageasset.New(imageasset.Options{
		BaseDir:    root,
		HTTPClient: r.httpClient,
		Limits: imageasset.Limits{
			MaxFetchedBytes:           maxAssetBytes + maxAssetBytes/2 + 1024,
			MaxEncodedBytes:           maxAssetBytes,
			MaxDecompressedBytes:      maxAssetBytes,
			MaxSVGBytes:               maxAssetBytes,
			MaxDecodedWidth:           hardMaxDimension,
			MaxDecodedHeight:          hardMaxDimension,
			MaxDecodedPixels:          hardMaxPixels,
			MaxAssets:                 256,
			MaxCumulativeEncodedBytes: 64 << 20,
			MaxCumulativeDecodedBytes: 256 << 20,
		},
	})
}

func bundledFont(name string) (*d2fonts.FontFamily, error) {
	if name == "" {
		return nil, nil
	}
	var family d2fonts.FontFamily
	switch strings.ToLower(strings.ReplaceAll(name, "_", "-")) {
	case "default", "source-sans", "source-sans-pro", "sourcesanspro":
		family = d2fonts.SourceSansPro
	case "mono", "source-code", "source-code-pro", "sourcecodepro":
		family = d2fonts.SourceCodePro
	case "hand-drawn", "handdrawn":
		family = d2fonts.HandDrawn
	default:
		return nil, fmt.Errorf("unsupported bundled family %q", name)
	}
	return &family, nil
}

func layoutResolver(options entity.LayoutOptions) (*string, func(string) (d2graph.LayoutGraph, error), error) {
	var selected *string
	if options.Engine != "" {
		engine := string(options.Engine)
		switch options.Engine {
		case entity.LayoutDagre, entity.LayoutELK, entity.LayoutTALA:
			selected = &engine
		default:
			return nil, nil, fmt.Errorf("unsupported layout %q; expected dagre, elk, or tala", options.Engine)
		}
	}
	resolver := func(engine string) (d2graph.LayoutGraph, error) {
		switch strings.ToLower(engine) {
		case "dagre":
			opts := d2dagrelayout.DefaultOpts
			if options.Dagre.NodeSeparation != 0 {
				opts.NodeSep = options.Dagre.NodeSeparation
			}
			if options.Dagre.EdgeSeparation != 0 {
				opts.EdgeSep = options.Dagre.EdgeSeparation
			}
			if opts.NodeSep <= 0 || opts.EdgeSep <= 0 {
				return nil, fmt.Errorf("dagre spacing must be positive")
			}
			return func(ctx context.Context, graph *d2graph.Graph) error { return d2dagrelayout.Layout(ctx, graph, &opts) }, nil
		case "elk":
			opts := d2elklayout.DefaultOpts
			if options.ELK.Algorithm != "" {
				opts.Algorithm = options.ELK.Algorithm
			}
			if options.ELK.NodeSpacing != 0 {
				opts.NodeSpacing = options.ELK.NodeSpacing
			}
			if options.ELK.Padding != "" {
				opts.Padding = options.ELK.Padding
			}
			if options.ELK.EdgeNodeSpacing != 0 {
				opts.EdgeNodeSpacing = options.ELK.EdgeNodeSpacing
			}
			if options.ELK.SelfLoopSpacing != 0 {
				opts.SelfLoopSpacing = options.ELK.SelfLoopSpacing
			}
			if opts.NodeSpacing <= 0 || opts.EdgeNodeSpacing <= 0 || opts.SelfLoopSpacing <= 0 {
				return nil, fmt.Errorf("ELK spacing must be positive")
			}
			return func(ctx context.Context, graph *d2graph.Graph) error { return d2elklayout.Layout(ctx, graph, &opts) }, nil
		case "tala":
			opts := d2talalayout.Options{Seeds: append([]int64(nil), options.TALA.Seeds...), MaxConcurrency: options.TALA.MaxConcurrency}
			if len(opts.Seeds) == 0 {
				opts.Seeds = nil
			}
			return func(ctx context.Context, graph *d2graph.Graph) error { return d2talalayout.Layout(ctx, graph, &opts) }, nil
		default:
			return nil, fmt.Errorf("unsupported layout %q; expected dagre, elk, or tala", engine)
		}
	}
	return selected, resolver, nil
}

func themeOverrides(values map[string]string) (*d2target.ThemeOverrides, error) {
	if len(values) == 0 {
		return nil, nil
	}
	overrides := &d2target.ThemeOverrides{}
	for key, value := range values {
		if value == "" {
			return nil, fmt.Errorf("theme token %s cannot be empty", key)
		}
		tokenValue := value
		switch strings.ToUpper(key) {
		case "N1":
			overrides.N1 = &tokenValue
		case "N2":
			overrides.N2 = &tokenValue
		case "N3":
			overrides.N3 = &tokenValue
		case "N4":
			overrides.N4 = &tokenValue
		case "N5":
			overrides.N5 = &tokenValue
		case "N6":
			overrides.N6 = &tokenValue
		case "N7":
			overrides.N7 = &tokenValue
		case "B1":
			overrides.B1 = &tokenValue
		case "B2":
			overrides.B2 = &tokenValue
		case "B3":
			overrides.B3 = &tokenValue
		case "B4":
			overrides.B4 = &tokenValue
		case "B5":
			overrides.B5 = &tokenValue
		case "B6":
			overrides.B6 = &tokenValue
		case "AA2":
			overrides.AA2 = &tokenValue
		case "AA4":
			overrides.AA4 = &tokenValue
		case "AA5":
			overrides.AA5 = &tokenValue
		case "AB4":
			overrides.AB4 = &tokenValue
		case "AB5":
			overrides.AB5 = &tokenValue
		default:
			return nil, fmt.Errorf("unknown theme token %q", key)
		}
	}
	return overrides, nil
}

func selectBoard(root *d2target.Diagram, path []string) (*d2target.Diagram, error) {
	if len(path) == 0 {
		return root, nil
	}
	board := root.GetBoard(path)
	if board == nil {
		return nil, fmt.Errorf("board %q was not found", strings.Join(path, "."))
	}
	return board, nil
}

func renderCompiled(ctx context.Context, root *d2target.Diagram, format entity.ExportFormat, renderOpts d2svg.RenderOpts, options normalizedRenderOptions, imageResolver *imageasset.Resolver) ([]byte, error) {
	includeDescendants := format == entity.FormatPDF || format == entity.FormatPPTX || format == entity.FormatGIF
	if options.Boards.IncludeDescendants != nil {
		includeDescendants = *options.Boards.IncludeDescendants
	}
	boards, err := collectBoards(ctx, root, includeDescendants, options.maxBoards)
	if err != nil {
		return nil, err
	}
	if len(boards) == 0 {
		return nil, fmt.Errorf("selected board tree has no renderable boards")
	}
	switch format {
	case entity.FormatSVG:
		return renderSVG(boards, root, renderOpts, options.Animation.IntervalMS)
	case entity.FormatPNG:
		if len(boards) != 1 {
			return nil, fmt.Errorf("PNG produces one image; select a board or disable descendants")
		}
		return renderPNG(ctx, boards[0].diagram, renderOpts, options, 2, 0, imageResolver)
	case entity.FormatASCII:
		if len(boards) != 1 {
			return nil, fmt.Errorf("ASCII produces one document; select a board or disable descendants")
		}
		return renderASCII(ctx, boards[0].diagram, renderOpts, options.ASCII)
	case entity.FormatPDF:
		return renderPDF(ctx, boards, renderOpts, options, imageResolver)
	case entity.FormatPPTX:
		return renderPPTX(ctx, boards, renderOpts, options, imageResolver)
	case entity.FormatGIF:
		return renderGIF(ctx, boards, renderOpts, options, imageResolver)
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

func collectBoards(ctx context.Context, root *d2target.Diagram, descendants bool, maxBoards int) ([]renderedBoard, error) {
	boards := make([]renderedBoard, 0, min(maxBoards, 8))
	active := make(map[*d2target.Diagram]bool)
	var walk func(*d2target.Diagram, string, []string, int) error
	walk = func(board *d2target.Diagram, id string, names []string, depth int) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if board == nil {
			return fmt.Errorf("board tree contains a nil board")
		}
		if depth > 64 {
			return fmt.Errorf("board tree exceeds depth 64")
		}
		if active[board] {
			return fmt.Errorf("board tree contains a cycle at %s", id)
		}
		active[board] = true
		defer delete(active, board)
		name := board.Name
		if name == "" {
			name = board.Root.Label
		}
		if name == "" {
			name = "root"
		}
		path := append(append([]string(nil), names...), name)
		if !board.IsFolderOnly {
			if len(boards) >= maxBoards {
				return fmt.Errorf("board count exceeds limit %d", maxBoards)
			}
			boards = append(boards, renderedBoard{diagram: board, id: id, names: path})
		}
		if !descendants {
			return nil
		}
		groups := []struct {
			name     string
			children []*d2target.Diagram
		}{{"layers", board.Layers}, {"scenarios", board.Scenarios}, {"steps", board.Steps}}
		for _, group := range groups {
			for _, child := range group.children {
				childName := ""
				if child != nil {
					childName = child.Name
				}
				if err := walk(child, id+"."+group.name+"."+childName, path, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := walk(root, "root", nil, 1); err != nil {
		return nil, err
	}
	return boards, nil
}

func singleBoard(board *d2target.Diagram) *d2target.Diagram {
	clone := *board
	clone.Layers = nil
	clone.Scenarios = nil
	clone.Steps = nil
	return &clone
}

func renderSVG(boards []renderedBoard, root *d2target.Diagram, options d2svg.RenderOpts, intervalMS int) ([]byte, error) {
	if len(boards) == 1 {
		return d2svg.Render(singleBoard(boards[0].diagram), &options)
	}
	if intervalMS == 0 {
		intervalMS = 1000
	}
	if intervalMS < 1 {
		return nil, fmt.Errorf("animated SVG interval must be positive")
	}
	noXML := true
	inner := options
	inner.NoXMLTag = &noXML
	svgs := make([][]byte, 0, len(boards))
	for _, board := range boards {
		svg, err := d2svg.Render(singleBoard(board.diagram), &inner)
		if err != nil {
			return nil, err
		}
		svgs = append(svgs, svg)
	}
	return d2animate.Wrap(root, svgs, options, intervalMS)
}

func renderASCII(ctx context.Context, board *d2target.Diagram, options d2svg.RenderOpts, asciiOptions entity.ASCIIOptions) ([]byte, error) {
	charsetType := charset.Unicode
	if asciiOptions.Charset == entity.ASCIIStandard {
		charsetType = charset.ASCII
	} else if asciiOptions.Charset != "" && asciiOptions.Charset != entity.ASCIIExtended {
		return nil, fmt.Errorf("unsupported ASCII charset %q", asciiOptions.Charset)
	}
	artist := d2ascii.NewASCIIartist()
	return artist.Render(ctx, singleBoard(board), &d2ascii.RenderOpts{Scale: options.Scale, Charset: charsetType})
}

func renderPNG(ctx context.Context, board *d2target.Diagram, options d2svg.RenderOpts, limits normalizedRenderOptions, deviceScale float64, timestamp time.Duration, imageResolver *imageasset.Resolver) ([]byte, error) {
	document, err := buildScene(ctx, board, options, imageResolver)
	if err != nil {
		return nil, err
	}
	frame, err := d2raster.Render(ctx, document, rasterOptions(limits, deviceScale, timestamp))
	if err != nil {
		return nil, fmt.Errorf("rasterize D2: %w", err)
	}
	return d2raster.EncodePNG(ctx, frame)
}

func buildScene(ctx context.Context, board *d2target.Diagram, options d2svg.RenderOpts, imageResolver *imageasset.Resolver) (*d2scene.Document, error) {
	return d2scenebuild.Build(ctx, singleBoard(board), d2scenebuild.Options{
		Pad: options.Pad, Scale: options.Scale, Center: options.Center,
		ThemeID: options.ThemeID, ThemeOverrides: options.ThemeOverrides,
		MaxNodes: 100_000, MaxPathCommands: 1_000_000,
		Sketch:       options.Sketch != nil && *options.Sketch,
		SketchBudget: d2scenebuild.SketchBudget{MaxOperationSets: 100_000, MaxOperations: 1_000_000, MaxPathCommands: 1_000_000},
		LinkBudget:   d2scenebuild.LinkBudget{MaxRegions: 4096, MaxStringBytes: 1 << 20},
		Appendix:     true,
		Assets: &d2scenebuild.AssetOptions{
			Resolver: imageResolver,
			SVGImportLimits: d2svgimport.Limits{
				MaxBytes: 10 << 20, MaxDepth: 256, MaxElements: 10_000, MaxAttributes: 20_000,
				MaxAttributeBytes: 10 << 20, MaxPathCommands: 100_000, MaxTransformFunctions: 10_000,
				MaxUseDepth: 128, MaxResources: 10_000,
			},
			SVGImportBudget: d2scenebuild.SVGImportBudget{
				MaxSourceBytes: 64 << 20, MaxElements: 100_000, MaxAttributes: 200_000,
				MaxAttributeBytes: 64 << 20, MaxPathCommands: 1_000_000, MaxTransformFunctions: 100_000,
				MaxDeclaredResources: 100_000, MaxExpandedUseInstances: 100_000,
			},
		},
	})
}

func rasterOptions(limits normalizedRenderOptions, scale float64, timestamp time.Duration) d2raster.FrameOptions {
	return d2raster.FrameOptions{
		Scale: scale, Time: timestamp, Background: color.White,
		MaxWidth: limits.maxWidth, MaxHeight: limits.maxHeight, MaxPixels: limits.maxPixels,
		MaxNodes: 100_000, MaxDepth: 128, MaxPathCommands: 1_000_000,
		MaxAnimationTracks: 100_000, MaxAnimationKeyframes: 1_000_000,
		MaxAssets: 256, MaxAssetBytes: 64 << 20, MaxDecodedAssetBytes: 256 << 20,
		MaxImportDepth: 32, MaxOffscreenBytes: 256 << 20,
		MaxEvenOddClipWork: 100_000_000, MaxScanlineWork: 500_000_000,
	}
}

func renderPDF(ctx context.Context, boards []renderedBoard, options d2svg.RenderOpts, limits normalizedRenderOptions, imageResolver *imageasset.Resolver) ([]byte, error) {
	document := pdf.Init()
	pageMap := make(map[string]int, len(boards))
	for index, board := range boards {
		pageMap[board.id] = index
	}
	for _, board := range boards {
		pngBytes, err := renderPNG(ctx, board.diagram, options, limits, 2, 0, imageResolver)
		if err != nil {
			return nil, fmt.Errorf("PDF board %s: %w", board.id, err)
		}
		titles := make([]pdf.BoardTitle, len(board.names))
		for i, name := range board.names {
			titles[i] = pdf.BoardTitle{Name: name, BoardID: board.id}
		}
		themeID := int64(0)
		if options.ThemeID != nil {
			themeID = *options.ThemeID
		}
		padding := int64(d2svg.DEFAULT_PADDING)
		if options.Pad != nil {
			padding = *options.Pad
		}
		if err := document.AddPDFPage(pngBytes, titles, themeID, board.diagram.Root.Fill, nil, padding, 0, 0, pageMap, len(boards) > 1); err != nil {
			return nil, fmt.Errorf("add PDF board %s: %w", board.id, err)
		}
	}
	writer := newBoundedWriter(limits.maxOutputBytes)
	if err := document.ExportTo(writer); err != nil {
		return nil, fmt.Errorf("encode PDF: %w", err)
	}
	return writer.Bytes(), nil
}

func renderPPTX(ctx context.Context, boards []renderedBoard, options d2svg.RenderOpts, limits normalizedRenderOptions, imageResolver *imageasset.Resolver) ([]byte, error) {
	presentation := pptx.NewPresentation("D2 diagram", "Generated by d2mcp", "D2 diagram", "d2mcp", "0.9.0", len(boards) > 1)
	pageMap := make(map[string]int, len(boards))
	for index, board := range boards {
		pageMap[board.id] = index
	}
	for _, board := range boards {
		pngBytes, err := renderPNG(ctx, board.diagram, options, limits, 2, 0, imageResolver)
		if err != nil {
			return nil, fmt.Errorf("PPTX board %s: %w", board.id, err)
		}
		titles := make([]pptx.BoardTitle, len(board.names))
		for i, name := range board.names {
			titles[i] = pptx.BoardTitle{Name: name, BoardID: board.id, LinkToSlide: pageMap[board.id] + 1}
		}
		if _, err := presentation.AddSlide(pngBytes, titles); err != nil {
			return nil, fmt.Errorf("add PPTX board %s: %w", board.id, err)
		}
	}
	writer := newBoundedWriter(limits.maxOutputBytes)
	if err := presentation.ExportTo(writer); err != nil {
		return nil, fmt.Errorf("encode PPTX: %w", err)
	}
	return writer.Bytes(), nil
}

func renderGIF(ctx context.Context, boards []renderedBoard, options d2svg.RenderOpts, limits normalizedRenderOptions, imageResolver *imageasset.Resolver) ([]byte, error) {
	framesPerBoard := 1
	intervalMS := limits.Animation.IntervalMS
	if intervalMS > 0 {
		var err error
		framesPerBoard, err = xgif.AnimationFrameCount(intervalMS)
		if err != nil {
			return nil, err
		}
	} else {
		intervalMS = 1000
	}
	totalFrames := framesPerBoard * len(boards)
	if totalFrames > limits.maxFrames {
		return nil, fmt.Errorf("GIF requires %d frames, exceeding limit %d", totalFrames, limits.maxFrames)
	}
	frames := make([]*image.Paletted, 0, totalFrames)
	maxWidth, maxHeight := 0, 0
	var totalPixels int64
	for _, board := range boards {
		for frameIndex := 0; frameIndex < framesPerBoard; frameIndex++ {
			timestamp := time.Duration(0)
			if limits.Animation.IntervalMS > 0 {
				var err error
				timestamp, err = xgif.AnimationFrameTime(frameIndex)
				if err != nil {
					return nil, err
				}
			}
			document, err := buildScene(ctx, board.diagram, options, imageResolver)
			if err != nil {
				return nil, fmt.Errorf("GIF board %s: %w", board.id, err)
			}
			frame, err := d2raster.Render(ctx, document, rasterOptions(limits, 1, timestamp))
			if err != nil {
				return nil, fmt.Errorf("GIF board %s frame %d: %w", board.id, frameIndex, err)
			}
			pixels := int64(frame.Bounds().Dx()) * int64(frame.Bounds().Dy())
			if pixels > maxAggregatePixels-totalPixels {
				return nil, fmt.Errorf("GIF frames exceed the %d-pixel aggregate limit", maxAggregatePixels)
			}
			totalPixels += pixels
			paletted, err := xgif.QuantizeImage(ctx, frame)
			if err != nil {
				return nil, fmt.Errorf("quantize GIF frame: %w", err)
			}
			frames = append(frames, paletted)
			maxWidth = max(maxWidth, frame.Bounds().Dx())
			maxHeight = max(maxHeight, frame.Bounds().Dy())
		}
	}
	if limits.Animation.IntervalMS == 0 {
		intervalMS *= len(frames)
	}
	return xgif.AnimateCenteredOpaquePalettedImagesWithLimit(ctx, frames, maxWidth, maxHeight, intervalMS, limits.maxOutputBytes)
}

type boundedWriter struct {
	buffer bytes.Buffer
	limit  int64
}

func newBoundedWriter(limit int64) *boundedWriter {
	return &boundedWriter{limit: limit}
}

func (w *boundedWriter) Write(data []byte) (int, error) {
	remaining := w.limit - int64(w.buffer.Len())
	if remaining <= 0 || int64(len(data)) > remaining {
		return 0, fmt.Errorf("encoded output exceeds %d-byte limit", w.limit)
	}
	return w.buffer.Write(data)
}

func (w *boundedWriter) Bytes() []byte {
	return w.buffer.Bytes()
}
