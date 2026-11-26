package costs

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "spun", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(Feature{}.Commands()...)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func rowFields(t *testing.T, out, prefix string) []string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, prefix+" ") {
			f := strings.Fields(line)
			return f[len(f)-3:]
		}
	}
	t.Fatalf("no row %q in:\n%s", prefix, out)
	return nil
}

func TestCommandTotals(t *testing.T) {
	out, err := runCommand(t, "costs", "totals", "testdata/bundle")
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range []string{"Category", "Level"} {
		if !strings.Contains(out, h+" ") {
			t.Errorf("missing %s header in:\n%s", h, out)
		}
	}
	if !strings.Contains(out, "\n\nLevel") {
		t.Errorf("levels table is not separated by a blank line:\n%s", out)
	}
	tests := []struct {
		row  string
		want [3]string
	}{
		{"Alpha", [3]string{"$150", "$260", "$370"}},
		{"Total to raise", [3]string{"$1,165.50", "$1,810", "$2,475"}},
		{"Plus adds", [3]string{"$55", "$90", "$115"}},
		{"Plus goal", [3]string{"$1,165.50", "$1,810", "$2,475"}},
		{"Base goal", [3]string{"$1,110.50", "$1,720", "$2,360"}},
	}
	for _, tt := range tests {
		got := rowFields(t, out, tt.row)
		if got[0] != tt.want[0] || got[1] != tt.want[1] || got[2] != tt.want[2] {
			t.Errorf("%s = %v, want %v", tt.row, got, tt.want)
		}
	}
	for _, row := range []string{"Startup subtotal", "Monthly operating", "Reserve", "Base adds"} {
		rowFields(t, out, row)
	}
}

func TestCommandNotEnabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "spun.yaml", "title: T\ndocuments:\n  - a.md\n")
	writeFile(t, dir, "a.md", "# A\n")
	_, err := runCommand(t, "costs", "totals", dir)
	if err == nil || !strings.Contains(err.Error(), `feature "costs" is not enabled in `) {
		t.Fatalf("err = %v, want not enabled", err)
	}

	writeFile(t, dir, "spun.yaml", "title: T\ndocuments:\n  - a.md\nfeatures:\n  costs: false\n")
	_, err = runCommand(t, "costs", "totals", dir)
	if err == nil || !strings.Contains(err.Error(), "is not enabled") {
		t.Fatalf("disabled: err = %v, want not enabled", err)
	}
}

func TestCommandInvalidData(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "spun.yaml", "title: T\ndocuments:\n  - a.md\nfeatures:\n  costs:\n    file: costs.yaml\n")
	writeFile(t, dir, "a.md", "# A\n")
	writeFile(t, dir, "costs.yaml", strings.Replace(exampleYAML, "low: 100,", "low: 400,", 1))
	_, err := runCommand(t, "costs", "totals", dir)
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{"costs.yaml", "low must not exceed expected"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err = %q, want %q", err, want)
		}
	}
}

func TestCommandMissingBundle(t *testing.T) {
	_, err := runCommand(t, "costs", "totals", t.TempDir()+"/missing")
	if err == nil {
		t.Fatal("want error")
	}
}
