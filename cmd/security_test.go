package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseWorkspaceRoots(t *testing.T) {
	defaultPath := filepath.Join("workspaces", "default")
	assetPath := filepath.Join("workspaces", "assets")
	gotDefault, gotNamed, err := parseWorkspaceRoots([]string{
		"default=" + defaultPath,
		"assets=" + assetPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotDefault != defaultPath || gotNamed["assets"] != assetPath {
		t.Fatalf("parseWorkspaceRoots() = %q, %#v", gotDefault, gotNamed)
	}
}

func TestParseWorkspaceRootsRejectsInvalidAndDuplicateAssignments(t *testing.T) {
	for _, values := range [][]string{{"missing-separator"}, {"name="}, {"a=one", "a=two"}} {
		if _, _, err := parseWorkspaceRoots(values); err == nil {
			t.Fatalf("parseWorkspaceRoots(%q) succeeded", values)
		}
	}
}

func TestConfigureSecurityUsesCurrentDirectoryByDefault(t *testing.T) {
	workspaces, client, err := configureSecurity(nil, 1024, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspaces.Root("default"); err != nil {
		t.Fatal(err)
	}
	if client.Timeout != 2*time.Second {
		t.Fatalf("client timeout = %v, want 2s", client.Timeout)
	}
}
