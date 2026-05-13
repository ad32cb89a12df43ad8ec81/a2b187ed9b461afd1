package main

import (
	"os"

	"github.com/spf13/cobra"
)

type Manifest struct {
	ChunkSizeBytes int64          `json:"chunk_size_bytes"`
	Files          []ManifestItem `json:"files"`
	PartsDir       string         `json:"parts-dir"`
}

type ManifestItem struct {
	Name string `json:"name"`
}

var rootCmd = &cobra.Command{
	Use:   "chunker",
	Short: "Split and reconstruct large files using JSON manifest + aria2c",
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newSplitCmd())
	rootCmd.AddCommand(newMergeCmd())
}
