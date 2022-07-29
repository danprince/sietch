package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"reflect"
	"testing"
)

func setup(t *testing.T, fileMap map[string]string) builder {
	tmpDir := path.Join("/tmp/", fmt.Sprintf("%x", rand.Intn(100_000)))

	for filePath, contents := range fileMap {
		absPath := path.Join(tmpDir, filePath)
		dir := path.Dir(absPath)
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(absPath, []byte(contents), 0777)
		if err != nil {
			log.Fatal(err)
		}
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return builder{
		rootDir:  tmpDir,
		pagesDir: tmpDir,
	}
}

func TestCrawlDirs(t *testing.T) {
	b := setup(t, map[string]string{
		"a/b/c.md": "# C",
		"_d/e.md":  "#E",
		"index.md": "root",
		"e.js":     "console.log('e')",
		"f/g.css":  "body {}",
	})

	b.scan()
	actualDirs := b.dirs
	expectDirs := []string{"/a", "/f", "/a/b"}

	if !reflect.DeepEqual(actualDirs, expectDirs) {
		t.Errorf("\nexpected dirs: %v\n  actual dirs: %v", expectDirs, actualDirs)
	}
}

func TestCrawlPages(t *testing.T) {
	b := setup(t, map[string]string{
		"a/b/c.md": "# C",
		"_d/e.md":  "# E",
		"index.md": "root",
		"e.js":     "console.log('e')",
		"f/g.css":  "body {}",
	})

	b.scan()

	expectPages := []Page{
		{
			path:    "/index.md",
			Name:    "index.md",
			Url:     "/",
			depth:   0,
			outPath: "/index.html",
		},
		{
			path:    "/a/b/c.md",
			Name:    "c.md",
			Url:     "/a/b/c.html",
			outPath: "/a/b/c.html",
			depth:   3,
		},
	}

	actualPages := make([]Page, len(b.pages))

	for i, page := range b.pages {
		actualPages[i] = *page
	}

	if !reflect.DeepEqual(expectPages, actualPages) {
		t.Errorf("\nexpected pages:%+v\n  actual pages:%+v", expectPages, actualPages)
	}
}

func TestCrawlPagesByDepth(t *testing.T) {
	b := setup(t, map[string]string{
		"a/b/c.md":     "# C",
		"a/b/index.md": "# D",
		"_d/e.md":      "# E",
		"index.md":     "# Root",
	})

	b.scan()

	expectPagesByDepth := [][]*Page{
		{b.pages[0]},
		{},
		{b.pages[2]},
		{b.pages[1]},
	}

	actualPagesByDepth := b.pagesByDepth

	if !reflect.DeepEqual(expectPagesByDepth, actualPagesByDepth) {
		t.Errorf("\nexpected pages: %+v\nactual pages: %+v", expectPagesByDepth, actualPagesByDepth)
	}
}

func TestCrawlAssets(t *testing.T) {
	b := setup(t, map[string]string{
		"a/b/c.md": "# C",
		"_d/e.md":  "# E",
		"index.md": "# Root",
		"e.js":     "console.log('e')",
		"f/g.css":  "body {}",
		"_h.js":    "console.log('h')",
	})

	b.scan()

	expectAssets := []Asset{
		{Path: "/e.js"},
		{Path: "/f/g.css"},
	}

	actualAssets := make([]Asset, len(b.assets))

	for i, asset := range b.assets {
		actualAssets[i] = *asset
	}

	if !reflect.DeepEqual(expectAssets, actualAssets) {
		t.Errorf("\nexpected assets: %+v\nactual assets: %+v", expectAssets, actualAssets)
	}
}
