package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/etwodev/yuck-html/pkg/parse"
	"github.com/rs/zerolog"
)

var log = zerolog.New(zerolog.ConsoleWriter{
	Out:        os.Stdout,
	TimeFormat: "2006-01-02T15:04:05",
}).With().Timestamp().Str("service", "main").Logger()

func main() {
	if len(os.Args) != 3 {
		log.Error().Msg("Usage: yuck-html <input-folder> <output-folder>")
		os.Exit(1)
	}

	if os.Args[1] == "" || os.Args[2] == "" {
		log.Error().Msg("Input and output directories cannot be empty")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create output directory")
		os.Exit(1)
	}

	err = filepath.Walk(inputDir, walk(inputDir, outputDir))
	if err != nil {
		log.Fatal().Err(err).Msgf("Error walking through directory: %s", inputDir)
	}
}

func walk(inputDir, outputDir string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			log.Warn().Msgf("Skipping directory: %s", path)
			return nil
		}

		if !strings.HasSuffix(info.Name(), ".html") {
			log.Warn().Msgf("Skipping non-HTML file: %s", path)
			return nil
		}

		log.Info().Msgf("Processing file: %s", path)

		input, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %v", path, err)
		}

		yuckOutput, err := parse.TranspileYuckFromHTML(string(input))
		if err != nil {
			return fmt.Errorf("transpilation failed for %s: %v", path, err)
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		outPath := filepath.Join(outputDir, strings.TrimSuffix(relPath, ".html")+".yuck")
		os.MkdirAll(filepath.Dir(outPath), 0755)

		err = os.WriteFile(outPath, []byte(yuckOutput), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %v", outPath, err)
		}

		fmt.Printf("Transpiled: %s → %s\n", path, outPath)
		return nil
	}
}
