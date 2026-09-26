package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const DefaultRoot = "default"

var (
	ErrUnknownRoot = errors.New("unknown workspace root")
	ErrTraversal   = errors.New("path traversal is not allowed")
	ErrPathEscape  = errors.New("path escapes workspace root")
)

// Roots resolves paths against a fixed set of canonical workspace roots.
type Roots struct {
	roots map[string]string
}

// New creates a root set. An empty defaultRoot uses the current working directory.
func New(defaultRoot string, namedRoots map[string]string) (*Roots, error) {
	if defaultRoot == "" {
		var err error
		defaultRoot, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get current working directory: %w", err)
		}
	}

	configured := make(map[string]string, len(namedRoots)+1)
	configured[DefaultRoot] = defaultRoot
	for name, path := range namedRoots {
		if name == "" {
			return nil, errors.New("workspace root name cannot be empty")
		}
		if name == DefaultRoot {
			return nil, fmt.Errorf("workspace root name %q is reserved", DefaultRoot)
		}
		if path == "" {
			return nil, fmt.Errorf("workspace root %q path cannot be empty", name)
		}
		configured[name] = path
	}

	roots := make(map[string]string, len(configured))
	for name, path := range configured {
		canonical, err := canonicalRoot(path)
		if err != nil {
			return nil, fmt.Errorf("configure workspace root %q: %w", name, err)
		}
		roots[name] = canonical
	}
	return &Roots{roots: roots}, nil
}

// Root returns the canonical absolute path configured for name.
func (r *Roots) Root(name string) (string, error) {
	root, ok := r.roots[name]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownRoot, name)
	}
	return root, nil
}

// FS returns a read-only file system confined to the named root.
func (r *Roots) FS(name string) (fs.FS, error) {
	if _, err := r.Root(name); err != nil {
		return nil, err
	}
	return rootFS{roots: r, name: name}, nil
}

// ResolveRead resolves an existing path and verifies its symlink-expanded path remains in the root.
func (r *Roots) ResolveRead(name, path string) (string, error) {
	root, candidate, err := r.candidate(name, path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve read path: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("make read path absolute: %w", err)
	}
	if !within(root, resolved) {
		return "", fmt.Errorf("%w: %q", ErrPathEscape, path)
	}
	return resolved, nil
}

// ResolveWrite resolves a possibly new path using its nearest existing ancestor and verifies it remains in the root.
func (r *Roots) ResolveWrite(name, path string) (string, error) {
	root, candidate, err := r.candidate(name, path)
	if err != nil {
		return "", err
	}

	ancestor := candidate
	var suffix []string
	for {
		_, statErr := os.Lstat(ancestor)
		if statErr == nil {
			break
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("inspect write path: %w", statErr)
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return "", fmt.Errorf("resolve write path: no existing ancestor")
		}
		suffix = append(suffix, filepath.Base(ancestor))
		ancestor = parent
	}

	resolved, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return "", fmt.Errorf("resolve write path: %w", err)
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		resolved = filepath.Join(resolved, suffix[i])
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("make write path absolute: %w", err)
	}
	if !within(root, resolved) {
		return "", fmt.Errorf("%w: %q", ErrPathEscape, path)
	}
	return resolved, nil
}

func (r *Roots) candidate(name, path string) (string, string, error) {
	root, err := r.Root(name)
	if err != nil {
		return "", "", err
	}
	if hasTraversal(path) {
		return "", "", fmt.Errorf("%w: %q", ErrTraversal, path)
	}

	var candidate string
	if filepath.IsAbs(path) {
		candidate = filepath.Clean(path)
	} else {
		candidate = filepath.Join(root, path)
	}
	candidate, err = filepath.Abs(candidate)
	if err != nil {
		return "", "", fmt.Errorf("make path absolute: %w", err)
	}
	if !within(root, candidate) {
		return "", "", fmt.Errorf("%w: %q", ErrPathEscape, path)
	}
	return root, candidate, nil
}

type rootFS struct {
	roots *Roots
	name  string
}

func (filesystem rootFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	resolved, err := filesystem.roots.ResolveRead(filesystem.name, filepath.FromSlash(name))
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	file, err := os.Open(resolved)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return file, nil
}

func canonicalRoot(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("make root absolute: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return "", errors.New("workspace root is not a directory")
	}
	return filepath.Clean(canonical), nil
}

func hasTraversal(path string) bool {
	for _, part := range strings.FieldsFunc(path, func(character rune) bool {
		return character <= 255 && os.IsPathSeparator(uint8(character))
	}) {
		if part == ".." {
			return true
		}
	}
	return false
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)))
}
