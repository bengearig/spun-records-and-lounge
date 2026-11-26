package costs

import (
	"strings"
	"testing"
)

func renderBlock(t *testing.T, data, body string) string {
	t.Helper()
	docs, err := assemble(t, data, "```costs\n"+body+"\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	return docs[0].HTML
}

func wantRows(t *testing.T, html string, rows ...string) {
	t.Helper()
	for _, r := range rows {
		if !strings.Contains(html, r+"\n") {
			t.Fatalf("html = %q, want row %q", html, r)
		}
	}
}

func rowCount(html string) int {
	return strings.Count(html, "<tr><td")
}

func TestBlockCategories(t *testing.T) {
	html := renderBlock(t, exampleYAML, "categories")
	want := `<table class="costs-table">
<thead>
<tr><th>Category</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th></tr>
</thead>
<tbody>
<tr><td>Alpha</td><td class="num">$150</td><td class="num">$260</td><td class="num">$370</td></tr>
<tr><td>Beta</td><td class="num">$1,000.50</td><td class="num">$1,500</td><td class="num">$2,000</td></tr>
<tr><td>Startup subtotal</td><td class="num">$1,150.50</td><td class="num">$1,760</td><td class="num">$2,370</td></tr>
<tr><td>Cash reserve</td><td class="num">$15</td><td class="num">$50</td><td class="num">$105</td></tr>
<tr><td>Total to raise</td><td class="num">$1,165.50</td><td class="num">$1,810</td><td class="num">$2,475</td></tr>
</tbody>
</table>
`
	if !strings.Contains(html, want) {
		t.Fatalf("html = %q, want containing %q", html, want)
	}
}

func TestBlockItems(t *testing.T) {
	const (
		a1 = `<tr><td>A one</td><td class="num">$100</td><td class="num">$200</td><td class="num">$300</td><td>estimate</td></tr>`
		a2 = `<tr><td>A two</td><td class="num">$50</td><td class="num">$60</td><td class="num">$70</td><td>estimate (owner input needed)</td></tr>`
		b1 = `<tr><td>B one</td><td class="num">$1,000.50</td><td class="num">$1,500</td><td class="num">$2,000</td><td>estimate</td></tr>`
	)
	head := `<tr><th>Item</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th><th>Status</th></tr>`

	html := renderBlock(t, exampleYAML, "items")
	if !strings.Contains(html, head+"\n") {
		t.Fatalf("html = %q, want header %q", html, head)
	}
	if !strings.Contains(html, a1+"\n"+a2+"\n"+b1+"\n</tbody>") {
		t.Fatalf("html = %q, want rows in category then file order", html)
	}

	html = renderBlock(t, exampleYAML, "items category=a")
	wantRows(t, html, a1, a2)
	if rowCount(html) != 2 {
		t.Fatalf("category=a: html = %q, want 2 rows", html)
	}

	html = renderBlock(t, exampleYAML, "items level=plus")
	wantRows(t, html, a2)
	if rowCount(html) != 1 {
		t.Fatalf("level=plus: html = %q, want 1 row", html)
	}

	html = renderBlock(t, exampleYAML, "items category=b level=plus")
	if rowCount(html) != 0 {
		t.Fatalf("category=b level=plus: html = %q, want no rows", html)
	}

	html = renderBlock(t, exampleYAML, "items source=false")
	if strings.Contains(html, "Source") {
		t.Fatalf("source=false: html = %q, want no Source column", html)
	}

	data := strings.Replace(exampleYAML, "name: A one,", "name: A one, source: Quote 12,", 1)
	html = renderBlock(t, data, "items source=true category=a")
	wantRows(t, html,
		`<tr><th>Item</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th><th>Status</th><th>Source</th></tr>`,
		`<tr><td>A one</td><td class="num">$100</td><td class="num">$200</td><td class="num">$300</td><td>estimate</td><td>Quote 12</td></tr>`,
		`<tr><td>A two</td><td class="num">$50</td><td class="num">$60</td><td class="num">$70</td><td>estimate (owner input needed)</td><td></td></tr>`,
	)
}

func TestBlockReserve(t *testing.T) {
	html := renderBlock(t, exampleYAML, "reserve")
	want := `<table class="costs-table">
<thead>
<tr><th>Item</th><th>Level</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th></tr>
</thead>
<tbody>
<tr><td>M one</td><td>Base</td><td class="num">$10</td><td class="num">$20</td><td class="num">$30</td></tr>
<tr><td>M two</td><td>Plus</td><td class="num">$5</td><td class="num">$5</td><td class="num">$5</td></tr>
<tr><td>Monthly total</td><td></td><td class="num">$15</td><td class="num">$25</td><td class="num">$35</td></tr>
<tr><td>Reserve months</td><td></td><td class="num">1</td><td class="num">2</td><td class="num">3</td></tr>
<tr><td>Reserve</td><td></td><td class="num">$15</td><td class="num">$50</td><td class="num">$105</td></tr>
</tbody>
</table>
`
	if !strings.Contains(html, want) {
		t.Fatalf("html = %q, want containing %q", html, want)
	}
}

func TestBlockEscapes(t *testing.T) {
	data := strings.Replace(exampleYAML, "name: A one,", `name: "<b>&", source: "<b>&",`, 1)
	html := renderBlock(t, data, "items source=true")
	if strings.Contains(html, "<b>") {
		t.Fatalf("html = %q, want no raw <b>", html)
	}
	if got := strings.Count(html, "&lt;b&gt;&amp;"); got != 2 {
		t.Fatalf("html = %q, want name and source escaped (got %d)", html, got)
	}
}

func TestBlockOther(t *testing.T) {
	md := "```go\ncategories\n```\n\n```costs extra\ncategories\n```\n"
	docs, err := assemble(t, exampleYAML, md)
	if err != nil {
		t.Fatal(err)
	}
	html := docs[0].HTML
	if strings.Contains(html, "costs-table") {
		t.Fatalf("html = %q, want no table", html)
	}
	for _, w := range []string{`class="language-go"`, `class="language-costs"`} {
		if !strings.Contains(html, w) {
			t.Fatalf("html = %q, want containing %q", html, w)
		}
	}
	if got := strings.Count(html, "<pre>"); got != 2 {
		t.Fatalf("html = %q, want 2 code blocks", html)
	}
}

func TestBlockErrors(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"empty", "", "empty block"},
		{"blank", "   ", "empty block"},
		{"more", "categories\nitems", "expected one line, got more"},
		{"unknown table", "bogus", `unknown table "bogus"`},
		{"unknown option", "categories x=1", `unknown option "x" for categories`},
		{"unknown option items", "items bogus=1", `unknown option "bogus" for items`},
		{"bad option", "items nope", `bad option "nope"`},
		{"unknown category", "items category=zzz", `unknown category "zzz"`},
		{"unknown level", "items level=nope", `unknown level "nope"`},
		{"source", "items source=maybe", "source must be true or false"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := assemble(t, exampleYAML, "one\n\n```costs\n"+tt.body+"\n```\n")
			if err == nil {
				t.Fatal("want error")
			}
			for _, w := range []string{"a.md", "line 3: costs block: " + tt.want} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("err = %q, want containing %q", err, w)
				}
			}
		})
	}
}

func TestStyles(t *testing.T) {
	var c configured
	css := c.Styles()
	for _, w := range []string{".costs-table .num", "text-align: right", "font-variant-numeric: tabular-nums"} {
		if !strings.Contains(css, w) {
			t.Fatalf("css = %q, want containing %q", css, w)
		}
	}
	if c.HeadHTML() != "" || c.Scripts() != "" {
		t.Fatal("want empty head and scripts")
	}
}
