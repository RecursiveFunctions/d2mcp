package catalog

import (
	"strings"
	"testing"
)

func TestListCategories(t *testing.T) {
	got := ListCategories()
	if len(got) != 6 {
		t.Fatalf("ListCategories() returned %d categories, want 6", len(got))
	}
	if got[0].ID != "language" || got[len(got)-1].ID != "output" {
		t.Fatalf("ListCategories() order = %q ... %q", got[0].ID, got[len(got)-1].ID)
	}

	got[0].Title = "changed"
	if ListCategories()[0].Title == "changed" {
		t.Fatal("ListCategories() exposed mutable catalog state")
	}
}

func TestListCategory(t *testing.T) {
	got := ListCategory("diagram-types")
	if len(got) != 3 {
		t.Fatalf("ListCategory(diagram-types) returned %d entries, want 3", len(got))
	}
	if got[0].ID != "structured-diagrams" || got[2].ID != "grid-diagrams" {
		t.Fatalf("ListCategory(diagram-types) order = %#v", got)
	}
	if got := ListCategory("Diagram-Types"); len(got) != 0 {
		t.Fatalf("ListCategory should require an exact category ID, got %#v", got)
	}
}

func TestLookupIsExactAndReturnsCopy(t *testing.T) {
	entry, ok := Lookup("layout-engines")
	if !ok {
		t.Fatal("Lookup(layout-engines) did not find entry")
	}
	if entry.CategoryID != "layout" || entry.URI != "d2://reference/v0.9.0/layout-engines" {
		t.Fatalf("Lookup(layout-engines) = %#v", entry)
	}
	if _, ok := Lookup("Layout-Engines"); ok {
		t.Fatal("Lookup should require an exact ID")
	}

	entry.SourceURLs[0] = "changed"
	fresh, _ := Lookup("layout-engines")
	if fresh.SourceURLs[0] == "changed" {
		t.Fatal("Lookup exposed mutable source URLs")
	}
}

func TestSearchRanksAndBoundsResults(t *testing.T) {
	results := Search("TALA", 5)
	if len(results) == 0 || results[0].ID != "layout-engines" {
		t.Fatalf("Search(TALA) first result = %#v", results)
	}

	results = Search("diagram", MaxSearchLimit+100)
	if len(results) > MaxSearchLimit {
		t.Fatalf("Search returned %d results, maximum is %d", len(results), MaxSearchLimit)
	}
	if results := Search("   ", 3); len(results) != 0 {
		t.Fatalf("empty Search returned %d results", len(results))
	}
	if results := Search("sequence stable", 0); len(results) != 1 || results[0].ID != "sequence-diagrams" {
		t.Fatalf("multi-term Search = %#v", results)
	}
}

func TestReadResource(t *testing.T) {
	entry, ok := Lookup("exports-v090")
	if !ok {
		t.Fatal("missing exports-v090 entry")
	}
	resource, ok := ReadResource(entry.URI)
	if !ok {
		t.Fatal("ReadResource did not find known URI")
	}
	if resource.URI != entry.URI || resource.MIMEType != "text/markdown; charset=utf-8" {
		t.Fatalf("ReadResource metadata = %#v", resource)
	}
	if !strings.Contains(resource.Text, "built-in renderer") || !strings.Contains(resource.Text, entry.SourceURLs[0]) {
		t.Fatal("ReadResource content is missing capability detail or source URL")
	}
	if _, ok := ReadResource("d2://reference/v0.9.0/missing"); ok {
		t.Fatal("ReadResource found unknown URI")
	}
}

func TestEveryEntryHasEmbeddedContent(t *testing.T) {
	for _, item := range entries {
		resource, ok := ReadResource(item.URI)
		if !ok {
			t.Errorf("ReadResource(%q) failed", item.URI)
			continue
		}
		if !strings.Contains(resource.Text, "Source") {
			t.Errorf("resource %q has no source section", item.ID)
		}
		for _, source := range item.SourceURLs {
			if !strings.Contains(resource.Text, source) {
				t.Errorf("resource %q omits source %q", item.ID, source)
			}
		}
	}
}
