package math

import (
	"bytes"
	"testing"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
	"github.com/yuin/goldmark"
)

func convert(t *testing.T, src string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(Extension)).Convert([]byte(src), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestParse(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"inline", "a $x^2$ b", `<p>a <span class="math inline" data-tex="x^2">x^2</span> b</p>` + "\n"},
		{"display one line", "$$E=mc^2$$", `<div class="math display" data-tex="E=mc^2">E=mc^2</div>` + "\n"},
		{"display multi-line", "$$\na+b\n=c\n$$", `<div class="math display" data-tex="a+b
=c">a+b
=c</div>` + "\n"},
		{"display inline in paragraph", "see $$x$$ here", `<p>see <div class="math display" data-tex="x">x</div> here</p>` + "\n"},
		{"display inline multi-line", "see $$x\n+y$$ here", `<p>see <div class="math display" data-tex="x
+y">x
+y</div> here</p>` + "\n"},
		{"price literal", "$5 and $10", "<p>$5 and $10</p>\n"},
		{"price no space", "costs $5, or $10.", "<p>costs $5, or $10.</p>\n"},
		{"escaped dollar", `cost \$5 and \$x\$`, "<p>cost $5 and $x$</p>\n"},
		{"dollar then space", "a $ b $ c", "<p>a $ b $ c</p>\n"},
		{"closing preceded by space", "a $x $ b", "<p>a $x $ b</p>\n"},
		{"unclosed display", "a $$ b", "<p>a $$ b</p>\n"},
		{"code span", "`$x$` and $y$", `<p><code>$x$</code> and <span class="math inline" data-tex="y">y</span></p>` + "\n"},
		{"fenced code", "```\n$x$\n$$\ny\n$$\n```", "<pre><code>$x$\n$$\ny\n$$\n</code></pre>\n"},
		{"indented code", "    $x$\n", "<pre><code>$x$\n</code></pre>\n"},
		{"escaping", "$a<b&c>d$", `<p><span class="math inline" data-tex="a&lt;b&amp;c&gt;d">a&lt;b&amp;c&gt;d</span></p>` + "\n"},
		{"escaping display block", "$$\na<b\n$$", `<div class="math display" data-tex="a&lt;b">a&lt;b</div>` + "\n"},
		{"emphasis", "*see $x_1$ now*", `<p><em>see <span class="math inline" data-tex="x_1">x_1</span> now</em></p>` + "\n"},
		{"underscores in math", "$a_b_c$", `<p><span class="math inline" data-tex="a_b_c">a_b_c</span></p>` + "\n"},
		{"list item", "- item $x$\n- $$y$$", `<ul>
<li>item <span class="math inline" data-tex="x">x</span></li>
<li>
<div class="math display" data-tex="y">y</div>
</li>
</ul>` + "\n"},
		{"block in list", "- a\n\n  $$\n  y\n  $$\n", `<ul>
<li>
<p>a</p>
<div class="math display" data-tex="y">y</div>
</li>
</ul>` + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convert(t, tt.in); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestFeatureRegistered(t *testing.T) {
	f, err := feature.Default.Lookup("math")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := f.(feature.MarkdownExtender)
	if !ok || len(m.Extenders()) == 0 {
		t.Fatalf("math feature is not a MarkdownExtender with extensions: %#v", f)
	}
}
