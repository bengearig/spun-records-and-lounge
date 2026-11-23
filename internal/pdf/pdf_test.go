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
	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("TMPDIR not empty after Print: %v", entries)
	}
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
