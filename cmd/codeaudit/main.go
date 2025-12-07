// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"flag"
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
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "analyze":
		if err := runAnalyze(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "report":
		if err := runReport(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "metrics":
		if err := runMetrics(os.Args[2:]); err != nil {
			log.Printf("error: %v", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage()
	default:
		log.Printf("unknown command %q\n", cmd)
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `codeaudit - static code quality analyzer

Usage:
  codeaudit analyze [options] [path]
  codeaudit report  [options] [path]
  codeaudit metrics

Commands:
  analyze   Analyze a source tree and persist a report under .codeaudit/report.json
  report    Render the last report (text or json)
  metrics   List supported metrics

Run "codeaudit <command> -h" for command-specific flags.
`)
}

func runAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	pathFlag := fs.String("path", ".", "Path to project root (can also be given as positional argument)")
	workersFlag := fs.Int("workers", 0, "Number of worker goroutines (0 = use NumCPU)")
	extsFlag := fs.String("ext", ".go,.c,.h,.cpp,.hpp", "Comma-separated list of file extensions to include")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root := *pathFlag
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}

	workers := *workersFlag
	if workers <= 0 {
		workers = runtime.NumCPU()
		if workers < 1 {
			workers = 1
		}
	}

	includeExt := parseExts(*extsFlag)

	scanner := infrastructure.NewFSScanner()
	storage := infrastructure.NewFileStorage()
	gitClient := gitadapter.NewGitCLI()

	parsers := []ports.CodeParser{
		parser.NewGoParser(),
		parser.NewCParser(),
	}

	uc := usecase.NewAnalyzeProjectUseCase(
		scanner,
		scanner,
		parsers,
		gitClient,
		storage,
		workers,
	)

	ctx := context.Background()
	report, err := uc.Execute(ctx, usecase.AnalyzeProjectRequest{
		RootPath:   root,
		IncludeExt: includeExt,
	})
	if err != nil {
		return err
	}

	rendererRegistry := outputadapter.NewRendererRegistry(
		outputadapter.NewTextRenderer(),
		outputadapter.NewJSONRenderer(),
	)
	textRenderer, ok := rendererRegistry.Get("text")
	if !ok {
		return fmt.Errorf("text renderer not registered")
	}

	out, err := textRenderer.Render(report)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func runReport(args []string) error {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	pathFlag := fs.String("path", ".", "Path to project root (can also be given as positional argument)")
	formatFlag := fs.String("format", "text", "Output format (text|json)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	root := *pathFlag
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}

	storage := infrastructure.NewFileStorage()
	rendererRegistry := outputadapter.NewRendererRegistry(
		outputadapter.NewTextRenderer(),
		outputadapter.NewJSONRenderer(),
	)
	uc := usecase.NewGenerateReportUseCase(storage, rendererRegistry)

	ctx := context.Background()
	out, err := uc.Execute(ctx, usecase.GenerateReportRequest{
		RootPath: root,
		Format:   *formatFlag,
	})
	if err != nil {
		return err
	}

	fmt.Println(out)
	return nil
}

func runMetrics(args []string) error {
	fs := flag.NewFlagSet("metrics", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	uc := usecase.NewListMetricsUseCase()
	ctx := context.Background()
	metrics := uc.Execute(ctx)

	fmt.Println("Supported metrics:")
	for _, m := range metrics {
		fmt.Printf("- [%s] %s (%s)\n    %s\n",
			m.Group, m.Name, m.ID, m.Description)
	}
	return nil
}

func parseExts(s string) []string {
	parts := strings.Split(s, ",")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exts = append(exts, p)
	}
	return exts
}
