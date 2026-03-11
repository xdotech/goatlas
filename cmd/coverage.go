package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/goatlas/goatlas/internal/config"
	"github.com/goatlas/goatlas/internal/coverage"
	"github.com/goatlas/goatlas/internal/db"
	"github.com/goatlas/goatlas/internal/indexer"
	"github.com/spf13/cobra"
)

var coverageFormat string
var coverageNoAI bool

var coverageCmd = &cobra.Command{
	Use:   "check-coverage <spec-file>",
	Short: "Check spec coverage against codebase",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := args[0]
		ctx := context.Background()

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		pool, err := db.NewPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return fmt.Errorf("connect db: %w", err)
		}
		defer pool.Close()

		// Parse spec file into feature sections
		sections, err := coverage.ParseSpecFile(specFile)
		if err != nil {
			return fmt.Errorf("parse spec: %w", err)
		}
		if len(sections) == 0 {
			return fmt.Errorf("no features found in spec file")
		}
		fmt.Printf("Found %d feature(s) in spec\n", len(sections))

		// Extract features using AI or regex fallback
		var features []coverage.Feature
		if !coverageNoAI && cfg.GeminiAPIKey != "" {
			parser, err := coverage.NewGeminiParser(ctx, cfg.GeminiAPIKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Gemini unavailable, using regex parser: %v\n", err)
				coverageNoAI = true
			} else {
				defer parser.Close()
				for _, sec := range sections {
					feat, err := parser.ExtractFeatureComponents(ctx, sec)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to extract %q: %v\n", sec.Name, err)
						feat = coverage.ParseWithRegex(sec)
					}
					features = append(features, *feat)
				}
			}
		}
		if coverageNoAI || cfg.GeminiAPIKey == "" {
			for _, sec := range sections {
				features = append(features, *coverage.ParseWithRegex(sec))
			}
		}

		// Detect implementation status for each feature
		indexerSvc := indexer.NewService(pool)
		detector := coverage.NewDetector(pool, indexerSvc.SymbolRepo, indexerSvc.EndpointRepo)

		var featureResults []coverage.FeatureResult
		for i := range features {
			result, err := detector.DetectFeature(ctx, &features[i])
			if err != nil {
				return fmt.Errorf("detect feature %q: %w", features[i].Name, err)
			}
			featureResults = append(featureResults, *result)
		}

		// Generate and render report
		report := coverage.GenerateReport(featureResults, specFile)

		switch coverageFormat {
		case "json":
			data, err := coverage.RenderJSON(report)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		case "md", "markdown":
			fmt.Println(coverage.RenderMarkdown(report))
		default:
			fmt.Println(coverage.RenderText(report))
		}
		return nil
	},
}

func init() {
	coverageCmd.Flags().StringVarP(&coverageFormat, "format", "f", "text", "Output format: text|json|md")
	coverageCmd.Flags().BoolVar(&coverageNoAI, "no-ai", false, "Use regex-only parsing (no Gemini API)")
}
