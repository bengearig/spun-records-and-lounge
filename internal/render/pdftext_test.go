package render

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	pdfObjRE     = regexp.MustCompile(`(?m)^(\d+) 0 obj\r?\n`)
	pdfKidsRE    = regexp.MustCompile(`(?s)/Kids \[(.*?)\]`)
	pdfRefRE     = regexp.MustCompile(`(\d+) 0 R`)
	pdfFontRE    = regexp.MustCompile(`/(F\d+) (\d+) 0 R`)
	pdfTextRE    = regexp.MustCompile(`/(F\d+) [\d.]+ Tf|<([0-9A-Fa-f]+)>`)
	pdfBfCharRE  = regexp.MustCompile(`<([0-9A-Fa-f]+)> <([0-9A-Fa-f]+)>`)
	pdfBfRangeRE = regexp.MustCompile(`<([0-9A-Fa-f]+)> <([0-9A-Fa-f]+)> <([0-9A-Fa-f]+)>`)
)

// pdfPageText returns the text of each page of a Chromium-printed PDF, with
// all whitespace removed because glyph runs are split arbitrarily.
func pdfPageText(t *testing.T, data []byte) []string {
	t.Helper()
	locs := pdfObjRE.FindAllSubmatchIndex(data, -1)
	objs := map[string][]byte{}
	for i, loc := range locs {
		end := len(data)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		objs[string(data[loc[2]:loc[3]])] = data[loc[1]:end]
	}
	stream := func(ref string) string {
		body := objs[ref]
		_, s, ok := bytes.Cut(body, []byte("stream\n"))
		if !ok {
			return ""
		}
		s = s[:bytes.LastIndex(s, []byte("endstream"))]
		if bytes.Contains(body, []byte("FlateDecode")) {
			r, err := zlib.NewReader(bytes.NewReader(s))
			if err != nil {
				t.Fatalf("inflate object %s: %v", ref, err)
			}
			out, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("inflate object %s: %v", ref, err)
			}
			return string(out)
		}
		return string(s)
	}
	cmap := func(font string) map[int]rune {
		m := map[int]rune{}
		tu := regexp.MustCompile(`/ToUnicode (\d+) 0 R`).FindSubmatch(objs[font])
		if tu == nil {
			return m
		}
		cm := stream(string(tu[1]))
		hex := func(s string) int { v, _ := strconv.ParseInt(s, 16, 32); return int(v) }
		for _, blk := range regexp.MustCompile(`(?s)beginbfrange(.*?)endbfrange`).FindAllStringSubmatch(cm, -1) {
			for _, r := range pdfBfRangeRE.FindAllStringSubmatch(blk[1], -1) {
				for c := hex(r[1]); c <= hex(r[2]); c++ {
					m[c] = rune(hex(r[3]) + c - hex(r[1]))
				}
			}
		}
		for _, blk := range regexp.MustCompile(`(?s)beginbfchar(.*?)endbfchar`).FindAllStringSubmatch(cm, -1) {
			for _, r := range pdfBfCharRE.FindAllStringSubmatch(blk[1], -1) {
				m[hex(r[1])] = rune(hex(r[2]))
			}
		}
		return m
	}

	var pages []string
	var walk func(ref string)
	walk = func(ref string) {
		body := objs[ref]
		if bytes.Contains(body, []byte("/Type /Pages")) {
			for _, k := range pdfRefRE.FindAllSubmatch(pdfKidsRE.FindSubmatch(body)[1], -1) {
				walk(string(k[1]))
			}
			return
		}
		fonts := map[string]map[int]rune{}
		for _, f := range pdfFontRE.FindAllSubmatch(body, -1) {
			fonts[string(f[1])] = cmap(string(f[2]))
		}
		var sb strings.Builder
		for _, c := range pdfRefRE.FindAllSubmatch(regexp.MustCompile(`/Contents (\d+ 0 R|\[[^\]]*\])`).FindSubmatch(body)[1], -1) {
			var cur map[int]rune
			for _, m := range pdfTextRE.FindAllStringSubmatch(stream(string(c[1])), -1) {
				if m[1] != "" {
					cur = fonts[m[1]]
					continue
				}
				for i := 0; i+4 <= len(m[2]); i += 4 {
					g, _ := strconv.ParseInt(m[2][i:i+4], 16, 32)
					if r, ok := cur[int(g)]; ok {
						sb.WriteRune(r)
					}
				}
			}
		}
		pages = append(pages, strings.Join(strings.Fields(sb.String()), ""))
	}
	catalog := regexp.MustCompile(`/Pages (\d+) 0 R`)
	for _, body := range objs {
		if bytes.Contains(body, []byte("/Type /Catalog")) {
			walk(string(catalog.FindSubmatch(body)[1]))
		}
	}
	if len(pages) == 0 {
		t.Fatal("no pages found in PDF")
	}
	return pages
}
