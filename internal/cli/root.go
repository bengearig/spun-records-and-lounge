package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/bengearig/spun-records-and-lounge/internal/feature"
)

var version = "dev"

func newRootCmd(reg *feature.Registry) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "spun",
		Short:         "Build the spun records and lounge business plan bundle",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE:          func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.SetVersionTemplate("spun {{.Version}}\n")
	cmd.AddCommand(newDocsCmd())
	for _, f := range reg.All() {
		if p, ok := f.(feature.CommandProvider); ok {
			cmd.AddCommand(p.Commands()...)
		}
	}
	return cmd
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCmd(feature.Default)
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	return 0
}

func Execute() int {
	return run(os.Args[1:], os.Stdout, os.Stderr)
}
