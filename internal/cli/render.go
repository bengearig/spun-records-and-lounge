package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var errRenderNotImplemented = errors.New("render is not implemented yet")

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render <bundle-dir>",
		Short: "Render a documentation bundle to PDF",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkBundleDir(args[0]); err != nil {
				return err
			}
			return errRenderNotImplemented
		},
	}
	cmd.Flags().StringP("output", "o", "", "output PDF path")
	cmd.Flags().String("chromium", "", "path to the Chromium executable")
	return cmd
}

func checkBundleDir(path string) error {
	info, err := os.Stat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("bundle directory %q does not exist", path)
	case err != nil:
		return fmt.Errorf("cannot access bundle directory %q: %w", path, err)
	case !info.IsDir():
		return fmt.Errorf("bundle path %q is a file, not a directory", path)
	}
	return nil
}
