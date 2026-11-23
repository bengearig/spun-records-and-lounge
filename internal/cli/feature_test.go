package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
)

type fakeFeature struct{}

func (fakeFeature) Name() string { return "fake" }
func (fakeFeature) Commands() []*cobra.Command {
	return []*cobra.Command{{
		Use:   "fakecmd",
		Short: "A fake feature command",
		Run:   func(cmd *cobra.Command, _ []string) { cmd.Println("fake ran") },
	}}
}

func execRoot(t *testing.T, reg *feature.Registry, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	cmd := newRootCmd(reg)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestFeatureCommand(t *testing.T) {
	reg := feature.NewRegistry()
	reg.Register(fakeFeature{})

	if help := execRoot(t, reg, "--help"); !strings.Contains(help, "fakecmd") {
		t.Fatalf("help missing fakecmd: %q", help)
	}
	if out := execRoot(t, reg, "fakecmd"); !strings.Contains(out, "fake ran") {
		t.Fatalf("fakecmd output = %q", out)
	}
}
