package costs

import (
	"strings"
	"testing"
)

func TestInline(t *testing.T) {
	tests := []struct{ md, want string }{
		{"{{costs:total}}", "$1,165.50 to $2,475"},
		{"{{costs:total.expected}}", "$1,810"},
		{"{{costs:startup.low}}", "$1,150.50"},
		{"{{costs:reserve.high}}", "$105"},
		{"{{costs:monthly.expected}}", "$25"},
		{"{{costs:category.b}}", "$1,000.50 to $2,000"},
		{"{{costs:item.a1.high}}", "$300"},
		{"{{costs:item.m1}}", "$10 to $30"},
		{"{{costs:item.m2}}", "$5"},
		{"{{costs:level.plus}}", "$55 to $115"},
		{"{{costs:level.plus.expected}}", "$90"},
		{"{{costs:goal.base}}", "$1,110.50 to $2,360"},
		{"{{costs:goal.plus.high}}", "$2,475"},
		{"# Cost {{costs:total.high}}", "Cost $2,475"},
		{"*{{costs:total.high}}*", "<em>$2,475</em>"},
	}
	for _, tt := range tests {
		t.Run(tt.md, func(t *testing.T) {
			docs, err := assemble(t, exampleYAML, tt.md+"\n")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(docs[0].HTML, tt.want) {
				t.Fatalf("html = %q, want containing %q", docs[0].HTML, tt.want)
			}
		})
	}
}

func TestInlineLiteral(t *testing.T) {
	tests := []struct{ name, md, want string }{
		{"escaped", `\{{costs:total}}`, "{{costs:total}}"},
		{"code span", "`{{costs:total}}`", "<code>{{costs:total}}</code>"},
		{"fenced", "```text\n{{costs:total}}\n```", "{{costs:total}}"},
		{"spaces", "{{ costs:total }}", "{{ costs:total }}"},
		{"unclosed", "{{costs:total", "{{costs:total"},
		{"other", "{{other}}", "{{other}}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs, err := assemble(t, exampleYAML, tt.md+"\n")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(docs[0].HTML, tt.want) || strings.Contains(docs[0].HTML, "$") {
				t.Fatalf("html = %q, want literal %q", docs[0].HTML, tt.want)
			}
		})
	}
}

func TestInlineErrors(t *testing.T) {
	tests := []struct{ path, want string }{
		{"item.nope", `unknown item "nope"`},
		{"level.nope", `unknown level "nope"`},
		{"goal.nope", `unknown level "nope"`},
		{"category.nope", `unknown category "nope"`},
		{"bogus", `unknown path "bogus"`},
		{"", "empty path"},
		{"total.huge", `unknown suffix "huge"`},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			_, err := assemble(t, exampleYAML, "one\n\ntext {{costs:"+tt.path+"}}\n")
			if err == nil {
				t.Fatal("want error")
			}
			for _, w := range []string{"a.md", "line 3", tt.want, "{{costs:" + tt.path + "}}"} {
				if !strings.Contains(err.Error(), w) {
					t.Fatalf("err = %q, want containing %q", err, w)
				}
			}
		})
	}
}
