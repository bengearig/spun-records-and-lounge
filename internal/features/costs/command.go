package costs

import (
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/bengearig/spun-records-and-lounge/internal/bundle"
	"github.com/bengearig/spun-records-and-lounge/internal/feature"
)

// Commands adds the costs command group.
func (Feature) Commands() []*cobra.Command {
	costs := &cobra.Command{
		Use:   Name,
		Short: "Work with the costs feature",
	}
	costs.AddCommand(&cobra.Command{
		Use:   "totals <bundle-dir>",
		Short: "Print the cost totals of a bundle",
		Args:  cobra.ExactArgs(1),
		RunE:  runTotals,
	})
	return []*cobra.Command{costs}
}

func runTotals(cmd *cobra.Command, args []string) error {
	b, err := bundle.Load(args[0])
	if err != nil {
		return err
	}
	fc, ok := b.Config.Features[Name]
	if !ok || !fc.Enabled {
		return fmt.Errorf("feature %q is not enabled in %s", Name, filepath.Join(b.Dir, bundle.ConfigFile))
	}
	name, err := DataFile(feature.BundleInfo{Dir: b.Dir, Options: fc.Options})
	if err != nil {
		return err
	}
	d, err := LoadIn(b.Dir, name)
	if err != nil {
		return err
	}
	t, err := d.Totals()
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	out := cmd.OutOrStdout()
	tw := newTable(out)
	writeRow(tw, "Category", "Low", "Expected", "High")
	for _, c := range t.Categories {
		writeRange(tw, c.Name, c.Range)
	}
	writeRange(tw, "Startup subtotal", t.Startup)
	writeRange(tw, "Monthly operating", t.Monthly)
	writeRange(tw, "Reserve", t.Reserve)
	writeRange(tw, "Total to raise", t.Total)
	if err := tw.Flush(); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	tw = newTable(out)
	writeRow(tw, "Level", "Low", "Expected", "High")
	for _, l := range t.Levels {
		writeRange(tw, l.Name+" adds", l.Adds)
		writeRange(tw, l.Name+" goal", l.Goal)
	}
	return tw.Flush()
}

func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

func writeRow(w io.Writer, cols ...string) {
	for i, c := range cols {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, c)
	}
	fmt.Fprintln(w)
}

func writeRange(w io.Writer, name string, r Range) {
	writeRow(w, name, r.Low.String(), r.Expected.String(), r.High.String())
}
