package pdf

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
)

var (
	objRE      = regexp.MustCompile(`(?m)^(\d+) 0 obj\r?\n`)
	pagesRefRE = regexp.MustCompile(`/Pages (\d+) 0 R`)
	destsRefRE = regexp.MustCompile(`/Dests (\d+) 0 R`)
	kidsRE     = regexp.MustCompile(`(?s)/Kids \[(.*?)\]`)
	refRE      = regexp.MustCompile(`(\d+) 0 R`)
	destRE     = regexp.MustCompile(`/((?:[^\s/\[\]<>()#]|#[0-9A-Fa-f]{2})+)\s*\[(\d+) 0 R`)
	nameEscRE  = regexp.MustCompile(`#([0-9A-Fa-f]{2})`)
)

// AnchorPages prints html and returns the 1-based page of each id.
func AnchorPages(ctx context.Context, exe string, html []byte, opts Options, ids []string) (map[string]int, error) {
	_, pages, err := PrintAnchors(ctx, exe, html, opts, ids)
	return pages, err
}

// PrintAnchors is Print plus the 1-based page of each id, read from the
// PDF's named destinations. Every id must be the target of a link in html;
// with no ids the PDF is not inspected.
func PrintAnchors(ctx context.Context, exe string, html []byte, opts Options, ids []string) ([]byte, map[string]int, error) {
	data, err := Print(ctx, exe, html, opts)
	if err != nil {
		return nil, nil, err
	}
	if len(ids) == 0 {
		return data, nil, nil
	}
	all, err := destinationPages(data)
	if err != nil {
		return nil, nil, err
	}
	pages := make(map[string]int, len(ids))
	for _, id := range ids {
		p, ok := all[id]
		if !ok {
			return nil, nil, fmt.Errorf("anchor %q not found in the printed PDF", id)
		}
		pages[id] = p
	}
	return data, pages, nil
}

// destinationPages maps each named destination to its 1-based page number.
func destinationPages(data []byte) (map[string]int, error) {
	locs := objRE.FindAllSubmatchIndex(data, -1)
	objs := make(map[string][]byte, len(locs))
	var catalog []byte
	for i, loc := range locs {
		end := len(data)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		body := data[loc[1]:end]
		objs[string(data[loc[2]:loc[3]])] = body
		if bytes.Contains(body, []byte("/Type /Catalog")) {
			catalog = body
		}
	}
	if catalog == nil {
		return nil, fmt.Errorf("read PDF anchors: no catalog")
	}
	root := pagesRefRE.FindSubmatch(catalog)
	if root == nil {
		return nil, fmt.Errorf("read PDF anchors: no page tree")
	}

	pageNum := map[string]int{}
	var walk func(ref string, depth int) error
	walk = func(ref string, depth int) error {
		body, ok := objs[ref]
		if !ok || depth > 16 {
			return fmt.Errorf("read PDF anchors: bad page tree node %s", ref)
		}
		if !bytes.Contains(body, []byte("/Type /Pages")) {
			pageNum[ref] = len(pageNum) + 1
			return nil
		}
		kids := kidsRE.FindSubmatch(body)
		if kids == nil {
			return fmt.Errorf("read PDF anchors: page tree node %s has no kids", ref)
		}
		for _, m := range refRE.FindAllSubmatch(kids[1], -1) {
			if err := walk(string(m[1]), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(string(root[1]), 0); err != nil {
		return nil, err
	}

	out := map[string]int{}
	d := destsRefRE.FindSubmatch(catalog)
	if d == nil {
		return out, nil
	}
	dests, ok := objs[string(d[1])]
	if !ok {
		return nil, fmt.Errorf("read PDF anchors: missing destinations object")
	}
	for _, m := range destRE.FindAllSubmatch(dests, -1) {
		if p, ok := pageNum[string(m[2])]; ok {
			name := decodeName(string(m[1]))
			out[name] = p
			// Chromium percent-encodes non-ASCII anchor names.
			if un, err := url.PathUnescape(name); err == nil {
				out[un] = p
			}
		}
	}
	return out, nil
}

func decodeName(s string) string {
	return string(nameEscRE.ReplaceAllFunc([]byte(s), func(m []byte) []byte {
		v, _ := strconv.ParseUint(string(m[1:]), 16, 8)
		return []byte{byte(v)}
	}))
}
