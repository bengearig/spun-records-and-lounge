package costs

import (
	"strings"
	"testing"
)

const (
	rangeHead = `<tr><th>Level</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th></tr>`
	levelHead = `<tr><th>Item</th><th class="num">Low</th><th class="num">Expected</th><th class="num">High</th><th>Status</th></tr>`
)

func numRow(name, lo, ex, hi string) string {
	return `<tr><td>` + name + `</td><td class="num">` + lo + `</td><td class="num">` + ex + `</td><td class="num">` + hi + `</td></tr>`
}

func levelRow(name, lo, ex, hi, status string) string {
	return strings.TrimSuffix(numRow(name, lo, ex, hi), "</tr>") + `<td>` + status + `</td></tr>`
}

func TestBlockGoals(t *testing.T) {
	html := renderBlock(t, exampleYAML, "goals")
	wantRows(t, html, rangeHead,
		numRow("Base", "$1,110.50", "$1,720", "$2,360"),
		numRow("Plus", "$1,165.50", "$1,810", "$2,475"),
	)
	if rowCount(html) != 2 {
		t.Fatalf("html = %q, want 2 rows", html)
	}
}

func TestBlockAdds(t *testing.T) {
	html := renderBlock(t, exampleYAML, "adds")
	wantRows(t, html, rangeHead,
		numRow("Base", "$1,110.50", "$1,720", "$2,360"),
		numRow("Plus", "$55", "$90", "$115"),
	)
	if rowCount(html) != 2 {
		t.Fatalf("html = %q, want 2 rows", html)
	}
}

func TestBlockLevelBase(t *testing.T) {
	html := renderBlock(t, exampleYAML, "level id=base")
	wantRows(t, html, levelHead)
	want := strings.Join([]string{
		levelRow("A one", "$100", "$200", "$300", "estimate"),
		levelRow("B one", "$1,000.50", "$1,500", "$2,000", "estimate"),
		levelRow("Items subtotal", "$1,100.50", "$1,700", "$2,300", ""),
		levelRow("Added reserve", "$10", "$20", "$60", ""),
		levelRow("Adds", "$1,110.50", "$1,720", "$2,360", ""),
		levelRow("Goal", "$1,110.50", "$1,720", "$2,360", ""),
	}, "\n")
	if !strings.Contains(html, want) {
		t.Fatalf("html = %q, want containing %q", html, want)
	}
}

func TestBlockLevelPlus(t *testing.T) {
	html := renderBlock(t, exampleYAML, "level id=plus")
	wantRows(t, html, levelHead)
	want := strings.Join([]string{
		levelRow("A two", "$50", "$60", "$70", "estimate (owner input needed)"),
		levelRow("Items subtotal", "$50", "$60", "$70", ""),
		levelRow("Added reserve", "$5", "$30", "$45", ""),
		levelRow("Adds", "$55", "$90", "$115", ""),
		levelRow("Goal", "$1,165.50", "$1,810", "$2,475", ""),
	}, "\n")
	if !strings.Contains(html, want) {
		t.Fatalf("html = %q, want containing %q", html, want)
	}
}

func TestBlockLevelEmpty(t *testing.T) {
	data := strings.Replace(exampleYAML, "level: plus, name: A two", "level: base, name: A two", 1)
	html := renderBlock(t, data, "level id=plus")
	if rowCount(html) != 4 {
		t.Fatalf("html = %q, want only the four total rows", html)
	}
	if !strings.Contains(html, levelRow("Items subtotal", "$0", "$0", "$0", "")) {
		t.Fatalf("html = %q, want zero items subtotal", html)
	}
}

func TestBlockLevelErrors(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"missing id", "level", `missing option "id" for level`},
		{"unknown level", "level id=nope", `unknown level "nope"`},
		{"goals option", "goals x=1", `unknown option "x" for goals`},
		{"adds option", "adds x=1", `unknown option "x" for adds`},
		{"level option", "level id=base x=1", `unknown option "x" for level`},
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
