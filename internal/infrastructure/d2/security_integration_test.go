package d2

import (
	"context"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/i2y/d2mcp/internal/domain/entity"
	"github.com/i2y/d2mcp/internal/security/remote"
	"github.com/i2y/d2mcp/internal/security/workspace"
)

func TestRepositoryConfinesImportsToSelectedWorkspace(t *testing.T) {
	root := securityScratchDir(t)
	outside := securityScratchDir(t)
	if err := os.WriteFile(filepath.Join(root, "shared.d2"), []byte("imported: Imported"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.d2"), []byte("secret: Secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	repository := secureTestRepository(t, root, nil)

	if err := repository.Create(context.Background(), &entity.Diagram{
		ID: "inside", Content: "...@shared", WorkspaceRoot: "assets",
	}); err != nil {
		t.Fatalf("confined import failed: %v", err)
	}
	if err := repository.Create(context.Background(), &entity.Diagram{
		ID: "escape", Content: "...@../" + filepath.Base(outside) + "/secret", WorkspaceRoot: "assets",
	}); err == nil {
		t.Fatal("traversal import succeeded")
	}
}

func TestRepositoryEmbedsConfinedLocalAndRemoteAssets(t *testing.T) {
	root := securityScratchDir(t)
	image := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"><rect width="1" height="1"/></svg>`)
	if err := os.WriteFile(filepath.Join(root, "logo.svg"), image, 0o600); err != nil {
		t.Fatal(err)
	}
	transport := &assetTransport{body: "GIF89a", contentType: "image/gif"}
	repository := secureTestRepository(t, root, &http.Client{Transport: transport})

	for _, test := range []struct {
		id   string
		icon string
		want string
	}{
		{id: "local", icon: "logo.svg", want: "data:image/svg+xml;base64,"},
		{id: "remote", icon: "https://assets.example/logo.gif", want: "data:image/gif;base64,"},
	} {
		content := `image: {shape: image; icon: ` + test.icon + `; width: 20; height: 20}`
		if err := repository.Create(context.Background(), &entity.Diagram{ID: test.id, Content: content, WorkspaceRoot: "assets"}); err != nil {
			t.Fatalf("Create(%s) error = %v", test.id, err)
		}
		reader, err := repository.Export(context.Background(), test.id, entity.FormatSVG)
		if err != nil {
			t.Fatalf("Export(%s) error = %v", test.id, err)
		}
		output, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(output), test.want) {
			t.Fatalf("Export(%s) did not embed %q", test.id, test.want)
		}
	}
	if transport.request == nil || transport.request.URL.String() != "https://assets.example/logo.gif" {
		t.Fatalf("remote asset request = %#v", transport.request)
	}

	pngReader, err := repository.Export(context.Background(), "local", entity.FormatPNG)
	if err != nil {
		t.Fatalf("PNG export with confined local asset: %v", err)
	}
	png, err := io.ReadAll(pngReader)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("PNG export header = %q", png)
	}
}

func TestRepositoryRejectsLocalAssetSymlinkEscape(t *testing.T) {
	root := securityScratchDir(t)
	outside := securityScratchDir(t)
	if err := os.WriteFile(filepath.Join(outside, "secret.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	repository := secureTestRepository(t, root, nil)
	content := `image: {shape: image; icon: escape/secret.svg; width: 20; height: 20}`
	if err := repository.Create(context.Background(), &entity.Diagram{ID: "escape", Content: content, WorkspaceRoot: "assets"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Export(context.Background(), "escape", entity.FormatSVG); err == nil {
		t.Fatal("symlink escape asset export succeeded")
	}
}

func secureTestRepository(t *testing.T, root string, client *http.Client) *D2Repository {
	t.Helper()
	workspaces, err := workspace.New(root, map[string]string{"assets": root})
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		client, err = remote.NewClient(remote.Config{Resolver: publicResolver{}})
		if err != nil {
			t.Fatal(err)
		}
	}
	created, err := NewD2RepositoryWithSecurity(SecurityConfig{
		Workspaces: workspaces, HTTPClient: client, MaxAssetBytes: 1 << 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.(*D2Repository)
}

type assetTransport struct {
	request     *http.Request
	body        string
	contentType string
}

func (transport *assetTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.request = request
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": {transport.contentType}},
		Body:       io.NopCloser(strings.NewReader(transport.body)),
		Request:    request,
	}, nil
}

type publicResolver struct{}

func (publicResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
}

func securityScratchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".security-integration-")
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
