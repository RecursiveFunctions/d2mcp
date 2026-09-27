package distribution

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const generatedHeader = "Generated from server.json and distribution/targets.yaml. Do not edit."

type Server struct {
	Name        string     `json:"name"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Version     string     `json:"version"`
	Repository  Repository `json:"repository"`
	Packages    []Package  `json:"packages"`
}

type Repository struct {
	URL string `json:"url"`
}

type Package struct {
	RegistryType     string            `json:"registryType"`
	Identifier       string            `json:"identifier"`
	RuntimeHint      string            `json:"runtimeHint"`
	Transport        Transport         `json:"transport"`
	RuntimeArguments []RuntimeArgument `json:"runtimeArguments"`
}

type Transport struct {
	Type string `json:"type"`
}

type RuntimeArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Targets struct {
	Publisher       string            `yaml:"publisher"`
	ExtensionID     string            `yaml:"extension_id"`
	Author          string            `yaml:"author"`
	Category        string            `yaml:"category"`
	WorkspaceTarget string            `yaml:"workspace_target"`
	Clients         map[string]string `yaml:"clients"`
}

type Generator struct {
	server  Server
	targets Targets
	image   string
}

type clientConfig struct {
	MCPServers map[string]serverConfig `json:"mcpServers"`
}

type serverConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type kiloManifest struct {
	ID            string          `yaml:"id"`
	Name          string          `yaml:"name"`
	Description   string          `yaml:"description"`
	Author        string          `yaml:"author"`
	URL           string          `yaml:"url"`
	Category      string          `yaml:"category"`
	Prerequisites []string        `yaml:"prerequisites"`
	Content       []kiloContent   `yaml:"content"`
	Parameters    []kiloParameter `yaml:"parameters"`
}

type kiloContent struct {
	Name          string   `yaml:"name"`
	Prerequisites []string `yaml:"prerequisites"`
	Content       string   `yaml:"content"`
}

type kiloParameter struct {
	Name        string `yaml:"name"`
	Key         string `yaml:"key"`
	Placeholder string `yaml:"placeholder"`
}

func Load(serverPath, targetsPath string) (*Generator, error) {
	serverData, err := os.ReadFile(serverPath)
	if err != nil {
		return nil, fmt.Errorf("read server metadata: %w", err)
	}
	targetData, err := os.ReadFile(targetsPath)
	if err != nil {
		return nil, fmt.Errorf("read target metadata: %w", err)
	}

	var server Server
	if err := json.Unmarshal(serverData, &server); err != nil {
		return nil, fmt.Errorf("decode server metadata: %w", err)
	}
	var targets Targets
	if err := yaml.Unmarshal(targetData, &targets); err != nil {
		return nil, fmt.Errorf("decode target metadata: %w", err)
	}

	generator := &Generator{server: server, targets: targets}
	if err := generator.validate(); err != nil {
		return nil, err
	}
	return generator, nil
}

func (g *Generator) validate() error {
	if g.server.Name == "" || g.server.Title == "" || g.server.Description == "" || g.server.Version == "" {
		return errors.New("server metadata requires name, title, description, and version")
	}
	if g.targets.Publisher == "" || g.targets.ExtensionID == "" || g.targets.Author == "" || g.targets.Category == "" {
		return errors.New("target metadata requires publisher, extension_id, author, and category")
	}
	if g.targets.WorkspaceTarget != "/workspace" {
		return fmt.Errorf("workspace_target must be /workspace, got %q", g.targets.WorkspaceTarget)
	}

	requiredClients := []string{"portable", "cursor", "windsurf", "trae", "antigravity", "zed", "kilo"}
	for _, client := range requiredClients {
		if strings.TrimSpace(g.targets.Clients[client]) == "" {
			return fmt.Errorf("workspace expression for %s is required", client)
		}
	}

	for _, pkg := range g.server.Packages {
		if pkg.RegistryType != "oci" {
			continue
		}
		if g.image != "" {
			return errors.New("server metadata must contain exactly one OCI package")
		}
		if pkg.RuntimeHint != "docker" || pkg.Transport.Type != "stdio" {
			return errors.New("OCI package must use Docker with STDIO transport")
		}
		if strings.HasSuffix(pkg.Identifier, ":latest") || !strings.HasSuffix(pkg.Identifier, ":"+g.server.Version) {
			return fmt.Errorf("OCI identifier %q must use immutable version %s", pkg.Identifier, g.server.Version)
		}
		mountFound := false
		for _, argument := range pkg.RuntimeArguments {
			if argument.Name == "--mount" && strings.Contains(argument.Value, "dst="+g.targets.WorkspaceTarget) {
				mountFound = true
			}
		}
		if !mountFound {
			return errors.New("OCI package must declare the confined workspace mount")
		}
		g.image = pkg.Identifier
	}
	if g.image == "" {
		return errors.New("server metadata must contain an OCI package")
	}
	return nil
}

func (g *Generator) Files() (map[string][]byte, error) {
	files := make(map[string][]byte)
	for _, client := range []string{"portable", "cursor", "windsurf", "trae", "antigravity"} {
		data, err := marshalJSON(g.client(g.targets.Clients[client]))
		if err != nil {
			return nil, err
		}
		files[filepath.Join("distribution", "generated", client, "mcp.json")] = data
	}
	zedConfig := map[string]any{"context_servers": map[string]serverConfig{
		"d2mcp": {Command: "docker", Args: g.args(g.targets.Clients["zed"])},
	}}
	zedData, err := marshalJSON(zedConfig)
	if err != nil {
		return nil, err
	}
	files[filepath.Join("distribution", "generated", "zed", "mcp.json")] = zedData

	portable := files[filepath.Join("distribution", "generated", "portable", "mcp.json")]
	files[".mcp.json"] = portable

	installPayload := struct {
		Name    string   `json:"name"`
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}{Name: "d2mcp", Command: "docker", Args: g.args(g.targets.Clients["portable"])}
	payload, err := json.Marshal(installPayload)
	if err != nil {
		return nil, err
	}
	files[filepath.Join("distribution", "generated", "vscode", "install-url.txt")] = []byte("vscode:mcp/install?" + url.QueryEscape(string(payload)) + "\n")
	files[filepath.Join("distribution", "generated", "trae", "install-url.txt")] = []byte("trae://trae.ai-ide/mcp-import?config=" + url.QueryEscape(string(files[filepath.Join("distribution", "generated", "trae", "mcp.json")])) + "\n")

	kiloConfig, err := marshalJSON(g.client(g.targets.Clients["kilo"]).MCPServers["d2mcp"])
	if err != nil {
		return nil, err
	}
	kilo := kiloManifest{
		ID: "d2mcp", Name: g.server.Title, Description: g.server.Description,
		Author: g.targets.Author, URL: g.server.Repository.URL, Category: g.targets.Category,
		Prerequisites: []string{"Docker"},
		Content:       []kiloContent{{Name: "Docker", Prerequisites: []string{"Docker"}, Content: string(kiloConfig)}},
		Parameters:    []kiloParameter{{Name: "Workspace path", Key: "WORKSPACE_PATH", Placeholder: "/absolute/path/to/workspace"}},
	}
	kiloData, err := yaml.Marshal(kilo)
	if err != nil {
		return nil, fmt.Errorf("encode Kilo manifest: %w", err)
	}
	files[filepath.Join("distribution", "generated", "kilo", "MCP.yaml")] = append([]byte("# "+generatedHeader+"\n"), kiloData...)

	cline := fmt.Sprintf("# %s\n\n## Cline Marketplace submission\n\n- Repository: %s\n- Registry: `%s`\n- Version: `%s`\n- Image: `%s`\n- Description: %s\n- Installation: Ask Cline to follow the Docker instructions in the repository README.\n- Verification: MCP initialize and tools/list must succeed with the workspace mounted at `%s`.\n",
		generatedHeader, g.server.Repository.URL, g.server.Name, g.server.Version, g.image, g.server.Description, g.targets.WorkspaceTarget)
	files[filepath.Join("distribution", "generated", "cline", "submission.md")] = []byte(cline)
	files[filepath.Join("integrations", "vscode", "src", "release.ts")] = []byte(fmt.Sprintf(
		"// %s\nexport const version = %q;\nexport const image = %q;\n", generatedHeader, g.server.Version, g.image,
	))

	manifest := map[string]any{
		"server": g.server.Name, "version": g.server.Version, "image": g.image,
		"publisher": g.targets.Publisher, "extensionId": g.targets.ExtensionID,
	}
	files[filepath.Join("distribution", "generated", "release.json")], err = marshalJSON(manifest)
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (g *Generator) Write(root string, check bool) error {
	files, err := g.Files()
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var stale []string
	for _, path := range paths {
		fullPath := filepath.Join(root, path)
		if check {
			existing, readErr := os.ReadFile(fullPath)
			if readErr != nil || !bytes.Equal(existing, files[path]) {
				stale = append(stale, path)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, files[path], 0o644); err != nil {
			return err
		}
	}
	if len(stale) > 0 {
		return fmt.Errorf("generated adapters are stale: %s", strings.Join(stale, ", "))
	}
	return nil
}

func (g *Generator) client(workspace string) clientConfig {
	return clientConfig{MCPServers: map[string]serverConfig{"d2mcp": {Command: "docker", Args: g.args(workspace)}}}
}

func (g *Generator) args(workspace string) []string {
	return []string{"run", "--rm", "-i", "--mount", "type=bind,src=" + workspace + ",dst=" + g.targets.WorkspaceTarget, g.image}
}

func marshalJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
