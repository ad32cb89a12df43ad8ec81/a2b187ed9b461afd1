package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/spf13/cobra"
)

func newMergeCmd() *cobra.Command {
	var (
		manifestPath string
		outputFile   string
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge parts to original file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if manifestPath == "" {
				return errors.New("--manifest is required")
			}

			raw, err := os.ReadFile(manifestPath)
			if err != nil {
				return err
			}

			var manifest Manifest
			if err := json.Unmarshal(raw, &manifest); err != nil {
				return err
			}

			if len(manifest.Files) == 0 {
				return errors.New("manifest has no files")
			}

			out, err := os.Create(outputFile)
			if err != nil {
				return err
			}
			defer out.Close()

			buf := make([]byte, 8<<20)

			for i, item := range manifest.Files {
				in, err := os.Open(path.Join(manifest.PartsDir, item.Name))
				if err != nil {
					return fmt.Errorf("missing part %d (%s): %w", i, item.Name, err)
				}

				if _, err := io.CopyBuffer(out, in, buf); err != nil {
					in.Close()
					return err
				}

				in.Close()
			}

			if err := out.Sync(); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "reconstructed: %s\n", outputFile)
			return nil
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "manifest", "m", "", "path to JSON manifest")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")

	return cmd
}
