package cli

import (
	"bytes"
	"strings"
	"testing"
)

func runCLI(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func TestRootVersion(t *testing.T) {
	code, out, _ := runCLI("--version")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out != "spun dev\n" {
		t.Fatalf("stdout = %q, want %q", out, "spun dev\n")
	}
}

func TestRootUnknownSubcommand(t *testing.T) {
	code, _, errOut := runCLI("bogus")
	if code == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if !strings.Contains(errOut, "unknown command") {
		t.Fatalf("stderr = %q, want unknown command error", errOut)
	}
	if strings.Contains(errOut, "Usage:") {
		t.Fatalf("stderr should not include usage: %q", errOut)
	}
}

func TestRootHelp(t *testing.T) {
	code, out, _ := runCLI("--help")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(out, "spun") {
		t.Fatalf("help output missing name: %q", out)
	}
}
