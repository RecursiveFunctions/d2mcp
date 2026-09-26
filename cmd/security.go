package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/i2y/d2mcp/internal/security/remote"
	"github.com/i2y/d2mcp/internal/security/workspace"
)

type workspaceRootFlags []string

func (values *workspaceRootFlags) String() string {
	return strings.Join(*values, ",")
}

func (values *workspaceRootFlags) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func configureSecurity(assignments []string, maxAssetBytes int64, timeout time.Duration) (*workspace.Roots, *http.Client, error) {
	defaultRoot, named, err := parseWorkspaceRoots(assignments)
	if err != nil {
		return nil, nil, err
	}
	workspaces, err := workspace.New(defaultRoot, named)
	if err != nil {
		return nil, nil, err
	}
	client, err := remote.NewClient(remote.Config{
		MaxResponseBytes: maxAssetBytes,
		RequestTimeout:   timeout,
	})
	if err != nil {
		return nil, nil, err
	}
	return workspaces, client, nil
}

func parseWorkspaceRoots(assignments []string) (string, map[string]string, error) {
	defaultRoot := ""
	named := make(map[string]string, len(assignments))
	seen := make(map[string]struct{}, len(assignments))
	for _, assignment := range assignments {
		name, path, ok := strings.Cut(assignment, "=")
		if !ok || name == "" || path == "" {
			return "", nil, fmt.Errorf("workspace root %q must use name=path", assignment)
		}
		if _, duplicate := seen[name]; duplicate {
			return "", nil, fmt.Errorf("workspace root %q is configured more than once", name)
		}
		seen[name] = struct{}{}
		if name == workspace.DefaultRoot {
			defaultRoot = path
		} else {
			named[name] = path
		}
	}
	return defaultRoot, named, nil
}
