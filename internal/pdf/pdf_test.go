package pdf

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeExe(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestFindPrecedence(t *testing.T) {
	dir := t.TempDir()
	onPath := fakeExe(t, dir, "chromium-browser")
	t.Setenv("PATH", dir)
	t.Setenv(EnvChromium, "")

	got, err := Find("")
	if err != nil || got != onPath {
		t.Fatalf("PATH lookup = %q, %v; want %q", got, err, onPath)
	}

	t.Setenv(EnvChromium, "/env/chromium")
	if got, _ := Find(""); got != "/env/chromium" {
		t.Errorf("env = %q; want /env/chromium", got)
	}
	if got, _ := Find("/flag/chromium"); got != "/flag/chromium" {
		t.Errorf("flag = %q; want /flag/chromium", got)
	}
}

func TestFindPathOrder(t *testing.T) {
	dir := t.TempDir()
	fakeExe(t, dir, "google-chrome")
	want := fakeExe(t, dir, "chromium")
	t.Setenv("PATH", dir)
	t.Setenv(EnvChromium, "")

	if got, err := Find(""); err != nil || got != want {
		t.Errorf("Find = %q, %v; want %q", got, err, want)
	}
}

func TestFindNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv(EnvChromium, "")

	_, err := Find("")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range append([]string{"SPUN_CHROMIUM", "--chromium", "PATH"}, candidates...) {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestPaperSize(t *testing.T) {
	if w, h, err := PaperSize("A4"); err != nil || w != 8.27 || h != 11.69 {
		t.Errorf("a4 = %v, %v, %v", w, h, err)
	}
	if w, h, err := PaperSize("letter"); err != nil || w != 8.5 || h != 11 {
		t.Errorf("letter = %v, %v, %v", w, h, err)
	}
	if _, _, err := PaperSize("legal"); err == nil {
		t.Error("expected error for unknown size")
	}
}

func chromium(t *testing.T) string {
	t.Helper()
	exe, err := Find("")
	if err != nil {
		t.Fatalf("chromium required: %v", err)
	}
	return exe
}

func letter() Options {
	w, h, _ := PaperSize("letter")
	return Options{PaperWidth: w, PaperHeight: h, Margins: Margins{1, 1, 1, 1}}
}

func TestPrintSimple(t *testing.T) {
	exe := chromium(t)
	opts := letter()
	opts.FooterTemplate = `<div style="font-size:8px"><span class="pageNumber"></span></div>`
	got, err := Print(context.Background(), exe, []byte("<html><body><h1>Hello</h1></body></html>"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF-: %q", got[:min(len(got), 8)])
	}
}

func TestPrintInlineImage(t *testing.T) {
	exe := chromium(t)
	html := `<html><body><img src="data:image/gif;base64,R0lGODlhAQABAAAAACw="></body></html>`
	got, err := Print(context.Background(), exe, []byte(html), letter())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}

func TestPrintWaitForFlag(t *testing.T) {
	exe := chromium(t)
	opts := letter()
	opts.WaitForFlag = "__ready"
	html := `<html><body><script>setTimeout(function(){window.__ready = true}, 200)</script></body></html>`
	got, err := Print(context.Background(), exe, []byte(html), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Error("output is not a PDF")
	}
}

func TestPrintTimeout(t *testing.T) {
	exe := chromium(t)
	opts := letter()
	opts.WaitForFlag = "__never"
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := Print(ctx, exe, []byte("<html></html>"), opts); err == nil {
		t.Fatal("expected error when context times out")
	}
}

func TestPrintLeavesTempDirEmpty(t *testing.T) {
	exe := chromium(t)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	if _, err := Print(context.Background(), exe, []byte("<html></html>"), letter()); err != nil {
		t.Fatal(err)
	}
	got, err := strayTempEntries(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("Print left entries in TMPDIR: %v", got)
	}
}

// strayTempEntries lists paths under dir other than the browser's own scratch entries (org.chromium.* and com.google.Chrome.*).
func strayTempEntries(dir string) ([]string, error) {
	var stray []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if strings.HasPrefix(d.Name(), "org.chromium.") || strings.HasPrefix(d.Name(), "com.google.Chrome.") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		stray = append(stray, rel)
		return nil
	})
	return stray, err
}

func TestStrayTempEntries(t *testing.T) {
	t.Run("ignores chromium scratch", func(t *testing.T) {
		dir := t.TempDir()
		fake := filepath.Join(dir, "org.chromium.Fake")
		if err := os.MkdirAll(filepath.Join(fake, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(fake, "sub", "page.html"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "org.chromium.Chromium.chromium_chrome_url_fetcher_.sAwtD6"), 0o755); err != nil {
			t.Fatal(err)
		}
		got, err := strayTempEntries(dir)
		if err != nil || len(got) != 0 {
			t.Errorf("got %v, %v; want none", got, err)
		}
	})

	t.Run("reports spun directory and html", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "spun-pdf-123"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "spun-pdf-123", "index.html"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "loose.html"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := strayTempEntries(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]bool{"spun-pdf-123": true, filepath.Join("spun-pdf-123", "index.html"): true, "loose.html": true}
		if len(got) != len(want) {
			t.Fatalf("got %v; want %v", got, want)
		}
		for _, g := range got {
			if !want[g] {
				t.Errorf("unexpected entry %q", g)
			}
		}
	})
}

func TestAnchorPages(t *testing.T) {
	exe := chromium(t)
	html := `<html><body><h1 id="top">Top</h1><a href="#second">go</a> <a href="#caf%C3%A9">c</a>` +
		`<div style="break-before:page"></div><h2 id="second">Second</h2>` +
		`<div style="break-before:page"></div><h2 id="café">Third</h2></body></html>`
	got, err := AnchorPages(context.Background(), exe, []byte(html), letter(), []string{"second", "café"})
	if err != nil {
		t.Fatal(err)
	}
	if got["second"] != 2 || got["café"] != 3 || len(got) != 2 {
		t.Errorf("pages = %v, want second:2 café:3", got)
	}
	if _, err := AnchorPages(context.Background(), exe, []byte(html), letter(), []string{"nope"}); err == nil {
		t.Error("expected error for an anchor that is not linked")
	}
}
