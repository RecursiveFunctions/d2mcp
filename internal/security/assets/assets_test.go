package assets

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i2y/d2mcp/internal/security/workspace"
)

func TestEmbedSVGImagesConfinesLocalAssets(t *testing.T) {
	root := scratchDir(t)
	outside := scratchDir(t)
	image := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`)
	if err := os.WriteFile(filepath.Join(root, "logo.svg"), image, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.svg"), image, 0o600); err != nil {
		t.Fatal(err)
	}
	workspaces, err := workspace.New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := New(workspaces, &http.Client{Transport: &staticTransport{}}, 1024)
	if err != nil {
		t.Fatal(err)
	}

	got, err := resolver.EmbedSVGImages(context.Background(), workspace.DefaultRoot, []byte(`<svg><image href="logo.svg"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), `href="data:image/svg+xml;base64,`) {
		t.Fatalf("embedded SVG = %s", got)
	}

	outsideReference := filepath.ToSlash(filepath.Join(outside, "secret.svg"))
	_, err = resolver.EmbedSVGImages(context.Background(), workspace.DefaultRoot, []byte(`<svg><image href="`+outsideReference+`"/></svg>`))
	if !errors.Is(err, workspace.ErrPathEscape) {
		t.Fatalf("outside asset error = %v, want ErrPathEscape", err)
	}
}

func TestEmbedSVGImagesFetchesHTTPSWithInjectedClient(t *testing.T) {
	root := scratchDir(t)
	workspaces, err := workspace.New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	transport := &staticTransport{body: "GIF89a", contentType: "image/gif"}
	resolver, err := New(workspaces, &http.Client{Transport: transport}, 1024)
	if err != nil {
		t.Fatal(err)
	}

	got, err := resolver.EmbedSVGImages(context.Background(), workspace.DefaultRoot, []byte(`<svg><image href="https://assets.example/logo.gif"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	if transport.request == nil || transport.request.URL.String() != "https://assets.example/logo.gif" {
		t.Fatalf("remote request = %#v", transport.request)
	}
	if !strings.Contains(string(got), `href="data:image/gif;base64,`) {
		t.Fatalf("embedded SVG = %s", got)
	}
}

func TestEmbedSVGImagesRejectsHTTPAndOversizedAssets(t *testing.T) {
	root := scratchDir(t)
	workspaces, err := workspace.New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := New(workspaces, &http.Client{Transport: &staticTransport{body: "GIF89a", contentType: "image/gif"}}, 4)
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolver.EmbedSVGImages(context.Background(), workspace.DefaultRoot, []byte(`<svg><image href="http://assets.example/logo.gif"/></svg>`))
	if !errors.Is(err, ErrUnsupportedAsset) {
		t.Fatalf("HTTP asset error = %v, want ErrUnsupportedAsset", err)
	}
	_, err = resolver.EmbedSVGImages(context.Background(), workspace.DefaultRoot, []byte(`<svg><image href="https://assets.example/logo.gif"/></svg>`))
	if !errors.Is(err, ErrAssetTooLarge) {
		t.Fatalf("oversized asset error = %v, want ErrAssetTooLarge", err)
	}
}

type staticTransport struct {
	request     *http.Request
	body        string
	contentType string
}

func (transport *staticTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.request = request
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": {transport.contentType}},
		Body:       io.NopCloser(strings.NewReader(transport.body)),
		Request:    request,
	}, nil
}

func scratchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".assets-test-")
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		os.RemoveAll(dir)
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(absolute) })
	return absolute
}
