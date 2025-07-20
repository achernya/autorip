package imdb

import (
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

type tmpDir struct {
	dir string
}

func newTmpDir(t *testing.T) *tmpDir {
	dir, err := os.MkdirTemp("", "autorip")
	if err != nil {
		t.Fatalf("failed to make tempdir: %+v", err)
	}
	return &tmpDir{
		dir: dir,
	}
}

func (t *tmpDir) Cleanup() {
	os.RemoveAll(t.dir) //nolint:errcheck
}

func TestCanMakeEmptyIndex(t *testing.T) {
	dir := newTmpDir(t)
	defer dir.Cleanup()
	index, err := NewIndex(dir.dir)
	if err != nil {
		t.Errorf("failed to open new index: %+v", err)
	}
	defer index.Close()
}

func TestCantOverwriteExistingIndex(t *testing.T) {
	dir := newTmpDir(t)
	defer dir.Cleanup()
	index, err := NewIndex(dir.dir)
	if err != nil {
		t.Errorf("failed to open new index: %+v", err)
	}
	index.Close()
	index, err = NewIndex(dir.dir)
	if err == nil {
		t.Error("unexpectedly succeeded in overwriting the index")
		index.Close()
	}
}

func copyTestData(dir string) error {
	for _, f := range desiredFiles {
		srcName := strings.Trim(f, filepath.Ext(f))
		src, err := os.Open(path.Join("testdata", srcName))
		if err != nil {
			return err
		}
		defer src.Close() //nolint:errcheck
		dst, err := os.Create(path.Join(dir, f))
		if err != nil {
			return err
		}
		defer dst.Close() //nolint:errcheck
		writer := gzip.NewWriter(dst)
		_, err = io.Copy(writer, src)
		if err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
	}
	return nil
}

func TestPopulateAndQueryIndex(t *testing.T) {
	dir := newTmpDir(t)
	defer dir.Cleanup()

	if err := copyTestData(dir.dir); err != nil {
		t.Fatalf("unable to prepare testdata: %+v", err)
	}

	index, err := NewIndex(dir.dir)
	if err != nil {
		t.Fatalf("failed to open new index: %+v", err)
	}

	if err := index.Build(); err != nil {
		t.Errorf("unable to build index: %+v", err)
	}
	index.Close()

	// Re-open the index read-only to make sure it can be queried.
	index, err = OpenIndex(dir.dir)
	if err != nil {
		t.Fatalf("failed to open existing index %+v", err)
	}
	defer index.Close()

	ch, err := index.Search(t.Context(), "Voice")
	if err != nil {
		t.Fatalf("error while performing search: %+v", err)
	}
	result, ok := <-ch
	if !ok {
		t.Fatal("got 0 results, want at least")
	}
	want := "tt0041069"
	if result.GetEntry().GetTConst() != want {
		t.Errorf("got %+q, want %+q", result.GetEntry().GetTConst(), want)
	}
	count := 0
	for {
		_, ok = <-ch
		if ok {
			count++
			continue
		}
		break
	}
	if count != 0 {
		t.Errorf("got %d unexpected extra results", count)
	}

	// Also double-check that JSON results work
	json, err := index.SearchJSON("Voice", 1)
	if err != nil {
		t.Errorf("couldn't get json-formatted search results: %+v", err)
	}
	if len(json) < 5 {
		t.Errorf("got %+q, want non-empty json", json)
	}
}
