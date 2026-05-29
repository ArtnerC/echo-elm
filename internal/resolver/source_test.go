package resolver_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/artnerc/echo-elm/internal/resolver"
)

func TestDirSource(t *testing.T) {
	dir := t.TempDir()
	if err := writeFile(t, dir, "FHIRHelpers.cql", "library FHIRHelpers"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(t, dir, "Foo-1.0.0.cql", "library Foo version '1.0.0'"); err != nil {
		t.Fatal(err)
	}
	src := resolver.NewDirSource(dir)

	t.Run("unversioned hit", func(t *testing.T) {
		b, ok, err := src.GetLibrarySource("FHIRHelpers", "")
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if !strings.Contains(string(b), "FHIRHelpers") {
			t.Errorf("unexpected content: %s", b)
		}
	})
	t.Run("versioned hit", func(t *testing.T) {
		b, ok, err := src.GetLibrarySource("Foo", "1.0.0")
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if !strings.Contains(string(b), "Foo") {
			t.Errorf("unexpected: %s", b)
		}
	})
	t.Run("versioned falls back to unversioned", func(t *testing.T) {
		_, ok, err := src.GetLibrarySource("FHIRHelpers", "4.0.1")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if !ok {
			t.Errorf("expected fallback hit")
		}
	})
	t.Run("miss", func(t *testing.T) {
		_, ok, err := src.GetLibrarySource("Nope", "")
		if err != nil {
			t.Fatalf("err=%v", err)
		}
		if ok {
			t.Errorf("unexpected hit")
		}
	})
}

func TestFSSource(t *testing.T) {
	fs := fstest.MapFS{
		"libs/Foo.cql":       &fstest.MapFile{Data: []byte("library Foo")},
		"libs/Bar-2.0.0.cql": &fstest.MapFile{Data: []byte("library Bar version '2.0.0'")},
	}
	src := resolver.NewFSSource(fs, "libs")

	if _, ok, err := src.GetLibrarySource("Foo", ""); err != nil || !ok {
		t.Errorf("Foo lookup: ok=%v err=%v", ok, err)
	}
	if _, ok, err := src.GetLibrarySource("Bar", "2.0.0"); err != nil || !ok {
		t.Errorf("Bar versioned: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := src.GetLibrarySource("Missing", ""); ok {
		t.Errorf("expected Missing miss")
	}

	// With empty dir
	rootSrc := resolver.NewFSSource(fstest.MapFS{
		"Foo.cql": &fstest.MapFile{Data: []byte("library Foo")},
	}, "")
	if _, ok, err := rootSrc.GetLibrarySource("Foo", ""); err != nil || !ok {
		t.Errorf("root Foo: ok=%v err=%v", ok, err)
	}
}

func TestMapSource(t *testing.T) {
	src := resolver.NewMapSource(map[string][]byte{
		"FHIRHelpers": []byte("library FHIRHelpers"),
		"Foo|1.0.0":   []byte("library Foo version '1.0.0'"),
		"Foo|2.0.0":   []byte("library Foo version '2.0.0'"),
	})

	if b, ok, _ := src.GetLibrarySource("FHIRHelpers", ""); !ok || string(b) != "library FHIRHelpers" {
		t.Errorf("FHIRHelpers lookup failed")
	}
	if b, ok, _ := src.GetLibrarySource("Foo", "2.0.0"); !ok || !strings.Contains(string(b), "2.0.0") {
		t.Errorf("versioned Foo lookup failed")
	}
	if _, ok, _ := src.GetLibrarySource("Missing", ""); ok {
		t.Errorf("Missing should not be found")
	}
	if _, ok, _ := src.GetLibrarySource("Foo", "9.9.9"); ok {
		t.Errorf("versioned miss should not fall back to unversioned-when-no-unversioned-key")
	}
}

func TestMultiSource(t *testing.T) {
	a := resolver.NewMapSource(map[string][]byte{"A": []byte("library A")})
	b := resolver.NewMapSource(map[string][]byte{"B": []byte("library B")})
	multi := resolver.NewMultiSource(a, b)

	if _, ok, _ := multi.GetLibrarySource("A", ""); !ok {
		t.Errorf("A in first source not found")
	}
	if _, ok, _ := multi.GetLibrarySource("B", ""); !ok {
		t.Errorf("B in second source not found")
	}
	if _, ok, _ := multi.GetLibrarySource("Missing", ""); ok {
		t.Errorf("missing should miss")
	}
}

func writeFile(t *testing.T, dir, name, content string) error {
	t.Helper()
	return writeAll(dir+"/"+name, content)
}
