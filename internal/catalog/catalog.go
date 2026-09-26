// Package catalog provides the embedded D2 v0.9.0 capability reference.
package catalog

import (
	"embed"
	"sort"
	"strings"
)

const (
	Version            = "v0.9.0"
	DefaultSearchLimit = 8
	MaxSearchLimit     = 20
	ResourceMIMEType   = "text/markdown; charset=utf-8"
	resourcePrefix     = "d2://reference/" + Version + "/"
)

// Category groups related D2 capabilities.
type Category struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// Entry describes one reference document in the catalog.
type Entry struct {
	ID         string   `json:"id"`
	CategoryID string   `json:"category_id"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	URI        string   `json:"uri"`
	SourceURLs []string `json:"source_urls"`
}

// ResourceContent is transport-neutral text suitable for an MCP resource response.
type ResourceContent struct {
	URI      string `json:"uri"`
	MIMEType string `json:"mimeType"`
	Text     string `json:"text"`
}

type embeddedEntry struct {
	Entry
	path string
}

//go:embed content/v0.9.0/*.md
var content embed.FS

var categories = []Category{
	{ID: "language", Title: "Language", Summary: "Core objects, containers, connections, and reusable definitions."},
	{ID: "visuals", Title: "Visuals", Summary: "Shapes, media, rich labels, styles, and themes."},
	{ID: "diagram-types", Title: "Diagram types", Summary: "Structured, sequence, and grid diagrams."},
	{ID: "composition", Title: "Composition", Summary: "Multi-board diagrams with layers, scenarios, and steps."},
	{ID: "layout", Title: "Layout", Summary: "Layout engines, direction, and engine-specific controls."},
	{ID: "output", Title: "Output", Summary: "Rendering and export capabilities."},
}

var entries = []embeddedEntry{
	newEntry("language-basics", "language", "Language basics", "Declare shapes, labels, containers, paths, and scoped references.", []string{
		"https://d2lang.com/tour/shapes/", "https://d2lang.com/tour/containers/", "https://d2lang.com/tour/connections/",
	}),
	newEntry("connections", "language", "Connections", "Connect objects with directed, undirected, repeated, styled, and indexed edges.", []string{
		"https://d2lang.com/tour/connections/",
	}),
	newEntry("reusing-definitions", "language", "Reusing definitions", "Reuse attributes and structure with classes, variables, and imports.", []string{
		"https://d2lang.com/tour/classes/", "https://d2lang.com/tour/imports/", "https://github.com/d2lang/d2/releases/tag/v0.9.0",
	}),
	newEntry("shapes-and-media", "visuals", "Shapes and media", "Choose built-in shapes and add remote or local icons and images.", []string{
		"https://d2lang.com/tour/shapes/", "https://d2lang.com/tour/icons/", "https://github.com/d2lang/d2/releases/tag/v0.9.0",
	}),
	newEntry("text-code-math", "visuals", "Text, code, and math", "Render Markdown, highlighted code, Unicode text, and MathJax notation.", []string{
		"https://d2lang.com/tour/text/", "https://github.com/d2lang/d2/releases/tag/v0.9.0",
	}),
	newEntry("styles-and-themes", "visuals", "Styles and themes", "Customize diagram appearance directly or with reusable classes and themes.", []string{
		"https://d2lang.com/tour/style/", "https://d2lang.com/tour/themes/",
	}),
	newEntry("structured-diagrams", "diagram-types", "Structured diagrams", "Build entity-relationship and UML class diagrams with dedicated shapes.", []string{
		"https://d2lang.com/tour/sql-tables/", "https://d2lang.com/tour/uml-classes/",
	}),
	newEntry("sequence-diagrams", "diagram-types", "Sequence diagrams", "Describe ordered actors, messages, groups, notes, and activation spans.", []string{
		"https://d2lang.com/tour/sequence-diagrams/", "https://github.com/d2lang/d2/releases/tag/v0.9.0",
	}),
	newEntry("grid-diagrams", "diagram-types", "Grid diagrams", "Arrange nested diagrams in rows and columns with explicit gaps and sizing.", []string{
		"https://d2lang.com/tour/grid-diagrams/",
	}),
	newEntry("composition", "composition", "Composition", "Create navigable layers, derived scenarios, and sequential steps.", []string{
		"https://d2lang.com/tour/composition/", "https://d2lang.com/tour/layers/", "https://d2lang.com/tour/scenarios/", "https://d2lang.com/tour/steps/",
	}),
	newEntry("layout-engines", "layout", "Layout engines", "Select and configure bundled Dagre, ELK, or open-source TALA.", []string{
		"https://d2lang.com/tour/layouts/", "https://github.com/d2lang/d2/releases/tag/v0.9.0",
	}),
	newEntry("exports-v090", "output", "Export capabilities in v0.9.0", "Render SVG, PNG, GIF, PDF, PPTX, and ASCII with v0.9.0's built-in renderers.", []string{
		"https://github.com/d2lang/d2/releases/tag/v0.9.0", "https://d2lang.com/tour/intro/",
	}),
}

func newEntry(id, categoryID, title, summary string, sources []string) embeddedEntry {
	return embeddedEntry{
		Entry: Entry{
			ID: id, CategoryID: categoryID, Title: title, Summary: summary,
			URI: resourcePrefix + id, SourceURLs: sources,
		},
		path: "content/v0.9.0/" + id + ".md",
	}
}

// ListCategories returns all categories in display order.
func ListCategories() []Category {
	return append([]Category(nil), categories...)
}

// ListCategory returns entries in display order for an exact category ID.
func ListCategory(categoryID string) []Entry {
	result := make([]Entry, 0)
	for _, item := range entries {
		if item.CategoryID == categoryID {
			result = append(result, cloneEntry(item.Entry))
		}
	}
	return result
}

// Lookup returns the entry whose ID exactly matches id.
func Lookup(id string) (Entry, bool) {
	for _, item := range entries {
		if item.ID == id {
			return cloneEntry(item.Entry), true
		}
	}
	return Entry{}, false
}

// Search returns at most limit entries matching every query term. A non-positive
// limit uses DefaultSearchLimit; values above MaxSearchLimit are clamped.
func Search(query string, limit int) []Entry {
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(terms) == 0 {
		return []Entry{}
	}
	limit = boundedLimit(limit)

	type scoredEntry struct {
		entry Entry
		score int
	}
	matches := make([]scoredEntry, 0)
	for _, item := range entries {
		body, err := content.ReadFile(item.path)
		if err != nil {
			continue
		}
		id := strings.ToLower(item.ID)
		title := strings.ToLower(item.Title)
		summary := strings.ToLower(item.Summary)
		category := strings.ToLower(item.CategoryID)
		searchable := id + " " + title + " " + summary + " " + category + " " + strings.ToLower(string(body))

		score := 0
		matched := true
		for _, term := range terms {
			if !strings.Contains(searchable, term) {
				matched = false
				break
			}
			switch {
			case title == term || id == term:
				score += 40
			case strings.Contains(title, term):
				score += 12
			case strings.Contains(id, term):
				score += 10
			case strings.Contains(summary, term):
				score += 5
			case strings.Contains(category, term):
				score += 3
			default:
				score++
			}
		}
		if matched {
			matches = append(matches, scoredEntry{entry: cloneEntry(item.Entry), score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			return matches[i].entry.ID < matches[j].entry.ID
		}
		return matches[i].score > matches[j].score
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}

	result := make([]Entry, len(matches))
	for i := range matches {
		result[i] = matches[i].entry
	}
	return result
}

// ReadResource returns embedded Markdown for an exact catalog URI.
func ReadResource(uri string) (ResourceContent, bool) {
	for _, item := range entries {
		if item.URI != uri {
			continue
		}
		body, err := content.ReadFile(item.path)
		if err != nil {
			return ResourceContent{}, false
		}
		return ResourceContent{URI: item.URI, MIMEType: ResourceMIMEType, Text: string(body)}, true
	}
	return ResourceContent{}, false
}

func boundedLimit(limit int) int {
	if limit <= 0 {
		return DefaultSearchLimit
	}
	if limit > MaxSearchLimit {
		return MaxSearchLimit
	}
	return limit
}

func cloneEntry(entry Entry) Entry {
	entry.SourceURLs = append([]string(nil), entry.SourceURLs...)
	return entry
}
