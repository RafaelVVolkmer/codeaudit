// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

// Command codeaudit provides a static code quality analyzer CLI.
//
// It exposes three subcommands:
//
//   - analyze: scan a source tree, compute metrics and persist a JSON report
//   - report:  render the last saved report in different formats
//   - metrics: list the available metric groups and identifiers
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	gitadapter "github.com/rafaelvolkmer/codeaudit/internal/adapter/git"
	outputadapter "github.com/rafaelvolkmer/codeaudit/internal/adapter/output"
	parser "github.com/rafaelvolkmer/codeaudit/internal/adapter/parser"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
	"github.com/rafaelvolkmer/codeaudit/internal/infrastructure"
	"github.com/rafaelvolkmer/codeaudit/internal/usecase"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// envPrefix defines the prefix used for environment variables that
	// configure the CLI. For example:
	//
	//   CODEAUDIT_PATH=/some/project
	//   CODEAUDIT_WORKERS=8
	envPrefix = "CODEAUDIT"
)

// App wires configuration, shared dependencies and command handlers for the CLI.
//
// It is intentionally small and focused on orchestration; all heavy lifting
// (scanning, parsing, metrics, persistence) is delegated to use cases and
// adapters in the internal packages.
type App struct {
	config *viper.Viper
	deps   *Dependencies
}

// Dependencies groups the shared services used by the CLI commands.
//
// By constructing these objects once and reusing them, the CLI avoids
// redundant wiring and makes it easier to test and evolve the application.
type Dependencies struct {
	Scanner     *infrastructure.FSScanner
	Storage     *infrastructure.FileStorage
	GitClient   ports.GitClient
	CodeParsers []ports.CodeParser
	Renderers   *outputadapter.RendererRegistry
}

// NewApp constructs a new App instance with a configured Viper instance
// and shared dependencies.
//
// Environment variables are configured with the CODEAUDIT_ prefix and
// hyphens in flag names are transparently mapped to underscores.
func NewApp() *App {
	config := viper.New()
	config.SetEnvPrefix(envPrefix)
	config.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	config.AutomaticEnv()

	deps := &Dependencies{
		Scanner:   infrastructure.NewFSScanner(),
		Storage:   infrastructure.NewFileStorage(),
		GitClient: gitadapter.NewGitCLI(),
		CodeParsers: []ports.CodeParser{
			parser.NewGoParser(),
			parser.NewCParser(),
			parser.NewCppParser(),
			parser.NewCSharpParser(),
		},
		Renderers: newRendererRegistry(),
	}

	return &App{
		config: config,
		deps:   deps,
	}
}

// main is the entry point for the CodeAudit CLI.
//
// It creates a root context, initializes the App and dispatches to the
// appropriate subcommand. All process exit codes are decided here.
func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	rootContext := context.Background()
	application := NewApp()

	command := os.Args[1]
	commandArgs := os.Args[2:]

	var err error

	switch command {
	case "analyze":
		err = application.runAnalyze(rootContext, commandArgs)
	case "report":
		err = application.runReport(rootContext, commandArgs)
	case "metrics":
		err = application.runMetrics(rootContext, commandArgs)
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		log.Printf("unknown command %q\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}

// printUsage prints the top-level usage text for the codeaudit CLI.
//
// It is intentionally concise and delegates command-specific help text
// to the individual subcommands.
func printUsage() {
	fmt.Fprintf(os.Stderr, `codeaudit - static code quality analyzer

Usage:
  codeaudit analyze [options] [path]
  codeaudit report  [options] [path]
  codeaudit metrics

Commands:
  analyze   Analyze a source tree and persist a report under .codeaudit/report.json
  report    Render the last report (text, json or sarif)
  metrics   List supported metrics

Run "codeaudit <command> -h" for command-specific flags.
`)
}

// runAnalyze handles the "analyze" subcommand.
//
// It scans the source tree, computes metrics, persists the report under
// .codeaudit/report.json and prints a human-readable or machine-readable
// summary to stdout.
//
// Configuration precedence (highest first):
//   1. Command-line flags
//   2. Environment variables CODEAUDIT_*
//   3. Built-in defaults
func (a *App) runAnalyze(ctx context.Context, args []string) error {
	flagSet := pflag.NewFlagSet("analyze", pflag.ContinueOnError)
	flagSet.SortFlags = false

	flagSet.String("path", ".", "Path to project root (can also be given as positional argument)")
	flagSet.Int("workers", 0, "Number of worker goroutines (0 = use NumCPU)")
	flagSet.String("ext", ".go,.c,.h,.cpp,.hpp,.cc,.hh,.cs", "Comma-separated list of file extensions to include")
	flagSet.String("format", "text", "Output format for immediate output (text|json|sarif)")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
  codeaudit analyze [options] [path]

Options:
`)
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	// Bind flags into the shared Viper instance so they can be overridden
	// by environment variables and still keep a single source of truth.
	if err := a.config.BindPFlags(flagSet); err != nil {
		return fmt.Errorf("bind flags to viper: %w", err)
	}

	rootPath := a.config.GetString("path")
	workerCount := a.config.GetInt("workers")
	extensionsValue := a.config.GetString("ext")
	outputFormat := a.config.GetString("format")

	// If the user provided a positional path argument, it wins over the flag.
	remainingArgs := flagSet.Args()
	if len(remainingArgs) > 0 {
		rootPath = remainingArgs[0]
	}

	if workerCount <= 0 {
		workerCount = runtime.NumCPU()
		if workerCount < 1 {
			workerCount = 1
		}
	}

	includeExtensions := parseExtensions(extensionsValue)

	analyzeUseCase := usecase.NewAnalyzeProjectUseCase(
		a.deps.Scanner,
		a.deps.Scanner,
		a.deps.CodeParsers,
		a.deps.GitClient,
		a.deps.Storage,
		workerCount,
	)

	projectReport, err := analyzeUseCase.Execute(ctx, usecase.AnalyzeProjectRequest{
		RootPath:   rootPath,
		IncludeExt: includeExtensions,
	})
	if err != nil {
		return err
	}

	renderer, found := a.deps.Renderers.Get(outputFormat)
	if !found {
		return fmt.Errorf("unknown format %q", outputFormat)
	}

	renderedOutput, err := renderer.Render(projectReport)
	if err != nil {
		return err
	}

	fmt.Println(renderedOutput)
	return nil
}

// runReport handles the "report" subcommand.
//
// It loads the last saved report from .codeaudit/report.json under the
// specified root directory and renders it in the requested format
// (text, json or sarif).
func (a *App) runReport(ctx context.Context, args []string) error {
	flagSet := pflag.NewFlagSet("report", pflag.ContinueOnError)
	flagSet.SortFlags = false

	flagSet.String("path", ".", "Path to project root (can also be given as positional argument)")
	flagSet.String("format", "text", "Output format (text|json|sarif)")

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
  codeaudit report [options] [path]

Options:
`)
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if err := a.config.BindPFlags(flagSet); err != nil {
		return fmt.Errorf("bind flags to viper: %w", err)
	}

	rootPath := a.config.GetString("path")
	outputFormat := a.config.GetString("format")

	remainingArgs := flagSet.Args()
	if len(remainingArgs) > 0 {
		rootPath = remainingArgs[0]
	}

	reportUseCase := usecase.NewGenerateReportUseCase(a.deps.Storage, a.deps.Renderers)

	renderedOutput, err := reportUseCase.Execute(ctx, usecase.GenerateReportRequest{
		RootPath: rootPath,
		Format:   outputFormat,
	})
	if err != nil {
		return err
	}

	fmt.Println(renderedOutput)
	return nil
}

// runMetrics handles the "metrics" subcommand.
//
// It currently has no flags and simply lists the available metric groups
// and identifiers to stdout.
func (a *App) runMetrics(ctx context.Context, args []string) error {
	flagSet := pflag.NewFlagSet("metrics", pflag.ContinueOnError)
	flagSet.SortFlags = false

	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage:
  codeaudit metrics

Lists the supported metric groups and identifiers.
`)
		flagSet.PrintDefaults()
	}

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	metricsUseCase := usecase.NewListMetricsUseCase()
	supportedMetrics := metricsUseCase.Execute(ctx)

	fmt.Println("Supported metrics:")
	for _, metric := range supportedMetrics {
		fmt.Printf("- [%s] %s (%s)\n    %s\n",
			metric.Group, metric.Name, metric.ID, metric.Description)
	}

	return nil
}

// parseExtensions normalizes a comma-separated list of file extensions into a
// slice of dot-prefixed extensions.
//
// Examples:
//
//	parseExtensions("go,c")        -> []string{".go", ".c"}
//	parseExtensions(".go,.c,.h")   -> []string{".go", ".c", ".h"}
func parseExtensions(raw string) []string {
	parts := strings.Split(raw, ",")
	var extensions []string

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, ".") {
			trimmed = "." + trimmed
		}
		extensions = append(extensions, trimmed)
	}

	return extensions
}

// newRendererRegistry constructs the default renderer registry used by the CLI.
//
// Keeping this logic in a helper avoids duplicating renderer wiring and makes
// it straightforward to add new output formats in the future.
func newRendererRegistry() *outputadapter.RendererRegistry {
	return outputadapter.NewRendererRegistry(
		outputadapter.NewTextRenderer(),
		outputadapter.NewJSONRenderer(),
		outputadapter.NewSarifRenderer(),
	)
}
