package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNewUsesCurrentDirectoryAsDefault(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	roots, err := New("", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := roots.Root(DefaultRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Root() = %q, want %q", got, want)
	}
}

func TestResolveReadAndWriteWithinNamedRoot(t *testing.T) {
	root := projectScratchDir(t)
	if err := os.Mkdir(filepath.Join(root, "existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "existing", "asset.txt")
	if err := os.WriteFile(file, []byte("asset"), 0o600); err != nil {
		t.Fatal(err)
	}

	roots, err := New(root, map[string]string{"assets": root})
	if err != nil {
		t.Fatal(err)
	}
	readPath, err := roots.ResolveRead("assets", filepath.Join("existing", "asset.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if readPath != file {
		t.Fatalf("ResolveRead() = %q, want %q", readPath, file)
	}
	writePath, err := roots.ResolveWrite("assets", filepath.Join("existing", "new", "asset.txt"))
	if err != nil {
		t.Fatal(err)
	}
	wantWrite := filepath.Join(root, "existing", "new", "asset.txt")
	if writePath != wantWrite {
		t.Fatalf("ResolveWrite() = %q, want %q", writePath, wantWrite)
	}

	absoluteRead, err := roots.ResolveRead("assets", file)
	if err != nil {
		t.Fatal(err)
	}
	if absoluteRead != file {
		t.Fatalf("absolute ResolveRead() = %q, want %q", absoluteRead, file)
	}
}

func TestResolveRejectsTraversalAndOutsideAbsolutePaths(t *testing.T) {
	root := projectScratchDir(t)
	outside := projectScratchDir(t)
	roots, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"../outside", "inside/../file"} {
		if _, err := roots.ResolveWrite(DefaultRoot, path); !errors.Is(err, ErrTraversal) {
			t.Errorf("ResolveWrite(%q) error = %v, want ErrTraversal", path, err)
		}
	}
	if _, err := roots.ResolveRead(DefaultRoot, filepath.Join(outside, "file")); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("outside absolute ResolveRead() error = %v, want ErrPathEscape", err)
	}
	if _, err := roots.ResolveRead("missing", "file"); !errors.Is(err, ErrUnknownRoot) {
		t.Fatalf("unknown root ResolveRead() error = %v, want ErrUnknownRoot", err)
	}
}

func TestResolveRejectsSymlinkEscapes(t *testing.T) {
	root := projectScratchDir(t)
	outside := projectScratchDir(t)
	outsideFile := filepath.Join(outside, "asset.txt")
	if err := os.WriteFile(outsideFile, []byte("asset"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	roots, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := roots.ResolveRead(DefaultRoot, filepath.Join("escape", "asset.txt")); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink read error = %v, want ErrPathEscape", err)
	}
	if _, err := roots.ResolveWrite(DefaultRoot, filepath.Join("escape", "new.txt")); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("symlink write error = %v, want ErrPathEscape", err)
	}
}

func TestFSConfinesReadsToRoot(t *testing.T) {
	root := projectScratchDir(t)
	outside := projectScratchDir(t)
	if err := os.WriteFile(filepath.Join(root, "inside.txt"), []byte("inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "outside.txt"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	roots, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	filesystem, err := roots.FS(DefaultRoot)
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.Open("inside.txt")
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	if _, err := filesystem.Open("../outside.txt"); err == nil {
		t.Fatal("FS.Open allowed traversal")
	}
	if _, err := filesystem.Open("escape/outside.txt"); !errors.Is(err, ErrPathEscape) {
		t.Fatalf("FS.Open symlink escape error = %v, want ErrPathEscape", err)
	}
}

func TestResolveAllowsSymlinksThatRemainWithinRoot(t *testing.T) {
	root := projectScratchDir(t)
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	roots, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	got, err := roots.ResolveWrite(DefaultRoot, filepath.Join("link", "new.txt"))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(target, "new.txt")
	if got != want {
		t.Fatalf("ResolveWrite() = %q, want %q", got, want)
	}
}

func projectScratchDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".workspace-test-")
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
