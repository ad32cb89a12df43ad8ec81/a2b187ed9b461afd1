package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newSplitCmd() *cobra.Command {
	var (
		inputFile  string
		chunkSizeB int64
	)

	cmd := &cobra.Command{
		Use:   "split",
		Short: "Split a large file into <= chunk-size parts and print JSON manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			if inputFile == "" {
				return errors.New("--input is required")
			}
			if chunkSizeB <= 0 {
				return errors.New("--chunk-size-bytes must be > 0")
			}

			_, filename := path.Split(inputFile)

			outDir := filename + "-parts"
			manifest, err := splitLargeFile(inputFile, outDir, chunkSizeB)
			if err != nil {
				return err
			}

			data, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				return err
			}
			data = append(data, '\n')

			return os.WriteFile(filename+".manifest.json", data, 0o644)
		},
	}

	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "input file to split")
	cmd.Flags().Int64Var(&chunkSizeB, "chunk-size-bytes", 1<<30, "maximum size of each part in bytes (default 1GiB)")

	return cmd
}

func splitLargeFile(inputFile, outDir string, chunkSizeB int64) (*Manifest, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	src, err := os.Open(inputFile)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	manifest := &Manifest{
		ChunkSizeBytes: chunkSizeB,
		PartsDir:       outDir,
		Files:          make([]ManifestItem, 0),
	}

	buf := make([]byte, 8<<20) // 8 MiB
	partIndex := 0

	for {
		tmp, err := os.CreateTemp(outDir, ".chunk-*")
		if err != nil {
			return nil, err
		}

		hasher := sha256.New()
		var written int64

		for written < chunkSizeB {
			remain := chunkSizeB - written
			readSize := len(buf)
			if int64(readSize) > remain {
				readSize = int(remain)
			}

			n, rerr := src.Read(buf[:readSize])
			if n > 0 {
				if _, werr := tmp.Write(buf[:n]); werr != nil {
					tmp.Close()
					_ = os.Remove(tmp.Name())
					return nil, werr
				}
				if _, herr := hasher.Write(buf[:n]); herr != nil {
					tmp.Close()
					_ = os.Remove(tmp.Name())
					return nil, herr
				}
				written += int64(n)
			}

			if rerr != nil {
				if errors.Is(rerr, io.EOF) {
					break
				}
				tmp.Close()
				_ = os.Remove(tmp.Name())
				return nil, rerr
			}

			if n == 0 {
				break
			}
		}

		if written == 0 {
			tmp.Close()
			_ = os.Remove(tmp.Name())
			break
		}

		if err := tmp.Close(); err != nil {
			_ = os.Remove(tmp.Name())
			return nil, err
		}

		sum := hex.EncodeToString(hasher.Sum(nil))
		finalName := sum + ".part"
		finalPath := filepath.Join(outDir, finalName)

		if err := os.Rename(tmp.Name(), finalPath); err != nil {
			_ = os.Remove(tmp.Name())
			return nil, err
		}

		manifest.Files = append(manifest.Files, ManifestItem{
			Name: finalName,
		})
		partIndex++

		if written < chunkSizeB {
			break
		}
	}

	return manifest, nil
}
