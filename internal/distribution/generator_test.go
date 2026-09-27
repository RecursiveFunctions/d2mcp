package distribution

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGeneratedAdaptersShareImmutableImage(t *testing.T) {
	generator := loadTestGenerator(t)
	files, err := generator.Files()
	if err != nil {
		t.Fatal(err)
	}

	for path, data := range files {
		if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, "MCP.yaml") || strings.HasSuffix(path, ".md") {
			if strings.Contains(string(data), ":latest") {
				t.Fatalf("%s contains a latest tag", path)
			}
		}
	}

	for _, client := range []string{"portable", "cursor", "windsurf", "trae", "antigravity"} {
		path := filepath.Join("distribution", "generated", client, "mcp.json")
		var config clientConfig
		if err := json.Unmarshal(files[path], &config); err != nil {
			t.Fatalf("decode %s: %v", path, err)
		}
		server, ok := config.MCPServers["d2mcp"]
		if !ok || len(server.Args) == 0 {
			t.Fatalf("%s does not contain mcpServers.d2mcp arguments: %s", path, files[path])
		}
		if !slices.Contains(server.Args, generator.image) {
			t.Fatalf("%s arguments do not contain image %q: %v", path, generator.image, server.Args)
		}
	}
}

func TestZedUsesContextServersFormat(t *testing.T) {
	generator := loadTestGenerator(t)
	files, err := generator.Files()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("distribution", "generated", "zed", "mcp.json")
	var config struct {
		ContextServers map[string]serverConfig `json:"context_servers"`
	}
	if err := json.Unmarshal(files[path], &config); err != nil {
		t.Fatal(err)
	}
	server, ok := config.ContextServers["d2mcp"]
	if !ok {
		t.Fatal("Zed adapter does not contain context_servers.d2mcp")
	}
	if !slices.Contains(server.Args, generator.image) {
		t.Fatalf("Zed arguments do not contain image %q: %v", generator.image, server.Args)
	}
	if server.Args[4] != "type=bind,src=/absolute/path/to/workspace,dst=/workspace" {
		t.Fatalf("unexpected Zed mount %q", server.Args[4])
	}
}

func TestVSCodeInstallURLRoundTrips(t *testing.T) {
	generator := loadTestGenerator(t)
	files, err := generator.Files()
	if err != nil {
		t.Fatal(err)
	}
	raw := strings.TrimSpace(string(files[filepath.Join("distribution", "generated", "vscode", "install-url.txt")]))
	parts := strings.SplitN(raw, "?", 2)
	if len(parts) != 2 || parts[0] != "vscode:mcp/install" {
		t.Fatalf("unexpected install URL %q", raw)
	}
	payload, err := url.QueryUnescape(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var config serverConfig
	if err := json.Unmarshal([]byte(payload), &config); err != nil {
		t.Fatal(err)
	}
	if config.Command != "docker" || config.Args[len(config.Args)-1] != generator.image {
		t.Fatalf("unexpected install payload: %+v", config)
	}
}

func TestLoadRejectsLatestImage(t *testing.T) {
	root := repositoryRoot(t)
	serverData, err := os.ReadFile(filepath.Join(root, "server.json"))
	if err != nil {
		t.Fatal(err)
	}
	serverData = []byte(strings.ReplaceAll(string(serverData), ":0.5.0", ":latest"))
	temporaryServer := filepath.Join(t.TempDir(), "server.json")
	if err := os.WriteFile(temporaryServer, serverData, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(temporaryServer, filepath.Join(root, "distribution", "targets.yaml")); err == nil {
		t.Fatal("Load accepted a latest image tag")
	}
}

func loadTestGenerator(t *testing.T) *Generator {
	t.Helper()
	root := repositoryRoot(t)
	generator, err := Load(filepath.Join(root, "server.json"), filepath.Join(root, "distribution", "targets.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return generator
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
