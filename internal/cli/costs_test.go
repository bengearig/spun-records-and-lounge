package cli

import (
	"strings"
	"testing"

	_ "github.com/bengearig/spun-records-and-lounge/internal/features/costs"
)

func TestCostsCommandCLI(t *testing.T) {
	code, out, stderr := runCLI("costs", "totals", "../features/costs/testdata/bundle")
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", code, stderr)
	}
	for _, want := range []string{"Total to raise", "Plus goal"} {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q:\n%s", want, out)
		}
	}
	if _, help, _ := runCLI("--help"); !strings.Contains(help, "costs") {
		t.Errorf("--help does not mention costs:\n%s", help)
	}
}
