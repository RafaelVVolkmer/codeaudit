// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package usecase

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type AnalyzeProjectRequest struct {
	RootPath   string
	IncludeExt []string
}

type AnalyzeProjectUseCase struct {
	scanner ports.SourceFileScanner
	reader  ports.FileReader
	parsers []ports.CodeParser
	git     ports.GitClient
	storage ports.ReportStorage
	workers int
}

func NewAnalyzeProjectUseCase(
	scanner ports.SourceFileScanner,
	reader ports.FileReader,
	parsers []ports.CodeParser,
	git ports.GitClient,
	storage ports.ReportStorage,
	workers int,
) *AnalyzeProjectUseCase {
	return &AnalyzeProjectUseCase{
		scanner: scanner,
		reader:  reader,
		parsers: parsers,
		git:     git,
		storage: storage,
		workers: workers,
	}
}

func (uc *AnalyzeProjectUseCase) Execute(ctx context.Context, req AnalyzeProjectRequest) (*model.ProjectReport, error) {
	if req.RootPath == "" {
		return nil, fmt.Errorf("root path is required")
	}
	if uc.workers <= 0 {
		uc.workers = runtime.NumCPU()
		if uc.workers < 1 {
			uc.workers = 1
		}
	}

	filesList, err := uc.scanner.Scan(ctx, req.RootPath, req.IncludeExt)
	if err != nil {
		return nil, fmt.Errorf("scan source files: %w", err)
	}
	if len(filesList) == 0 {
		return nil, fmt.Errorf("no source files found under %s", req.RootPath)
	}

	jobs := make(chan string)
	results := make(chan *model.FileMetrics)
	errCh := make(chan error, len(filesList))

	var wg sync.WaitGroup
	for i := 0; i < uc.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				src, err := uc.reader.ReadFile(path)
				if err != nil {
					errCh <- fmt.Errorf("read %s: %w", path, err)
					continue
				}

				parser := uc.selectParser(path)
				if parser == nil {
					continue
				}

				fm, err := parser.ParseFile(path, src)
				if err != nil {
					errCh <- fmt.Errorf("parse %s: %w", path, err)
					continue
				}

				results <- fm
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, path := range filesList {
			jobs <- path
		}
	}()

	go func() {
		wg.Wait()
		close(results)
		close(errCh)
	}()

	var files []model.FileMetrics
	for fm := range results {
		if fm != nil {
			files = append(files, *fm)
		}
	}

	var warnings []string
	for e := range errCh {
		if e != nil {
			warnings = append(warnings, e.Error())
		}
	}

	gitMetrics, err := uc.git.CollectFileMetrics(ctx, req.RootPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("git metrics disabled: %v", err))
	}

	if gitMetrics != nil {
		for i := range files {
			p := files[i].Path
			if gm, ok := gitMetrics[p]; ok {
				files[i].Git = gm
				continue
			}
			if rel, err := filepath.Rel(req.RootPath, p); err == nil {
				if gm, ok := gitMetrics[rel]; ok {
					files[i].Git = gm
				}
			}
		}
	}

	report := buildProjectReport(req.RootPath, files, warnings)

	if err := uc.storage.Save(ctx, req.RootPath, report); err != nil {
		return nil, fmt.Errorf("save report: %w", err)
	}
	return report, nil
}

func (uc *AnalyzeProjectUseCase) selectParser(path string) ports.CodeParser {
	for _, p := range uc.parsers {
		if p.SupportsFile(path) {
			return p
		}
	}
	return nil
}

func buildProjectReport(root string, files []model.FileMetrics, warnings []string) *model.ProjectReport {
	var proj model.ProjectMetrics

	proj.TotalFiles = len(files)

	var sizes []int
	var totalCCN int
	var maxCCN int
	var totalFunctions int
	var functionsCcnGt10 int
	var functionsCcnGt20 int
	var fnGt50, fnGt80, fnGt100 int
	var paramsGe5 int
	var sumParams float64

	var sumCommentDensity float64
	var filesWithComments int

	var gitLinesAdded, gitLinesDeleted, gitCommits int

	for _, f := range files {
		proj.TotalFunctions += len(f.Functions)
		totalFunctions += len(f.Functions)
		totalCCN += f.Summary.CCNTotal

		if f.Summary.CCNMaxFunction > maxCCN {
			maxCCN = f.Summary.CCNMaxFunction
		}
		functionsCcnGt10 += f.Summary.FunctionsCCNGt10
		functionsCcnGt20 += f.Summary.FunctionsCCNGt20

		if f.Comments.TotalLines > 0 {
			sumCommentDensity += f.Comments.CommentDensity
			filesWithComments++
		}

		if f.Git != nil {
			gitLinesAdded += f.Git.LinesAdded
			gitLinesDeleted += f.Git.LinesDeleted
			gitCommits += f.Git.Commits
		}

		for _, fn := range f.Functions {
			sizes = append(sizes, fn.NLOC)
			sumParams += float64(fn.Parameters)
			if fn.NLOC > 50 {
				fnGt50++
			}
			if fn.NLOC > 80 {
				fnGt80++
			}
			if fn.NLOC > 100 {
				fnGt100++
			}
			if fn.Parameters >= 5 {
				paramsGe5++
			}
		}
	}

	proj.MaxCCNPerFunction = maxCCN
	if totalFunctions > 0 {
		proj.AvgCCNPerFunction = float64(totalCCN) / float64(totalFunctions)
		proj.FunctionsCCNGt10Pct = float64(functionsCcnGt10) / float64(totalFunctions)
		proj.FunctionsCCNGt20Pct = float64(functionsCcnGt20) / float64(totalFunctions)
		proj.AvgParamsPerFunction = sumParams / float64(totalFunctions)
	}
	proj.FunctionsGt50Lines = fnGt50
	proj.FunctionsGt80Lines = fnGt80
	proj.FunctionsGt100Lines = fnGt100
	proj.FunctionsParamsGe5 = paramsGe5

	if filesWithComments > 0 {
		proj.CommentDensityAvg = sumCommentDensity / float64(filesWithComments)
	}

	proj.GitTotalLinesAdded = gitLinesAdded
	proj.GitTotalLinesDeleted = gitLinesDeleted
	proj.GitTotalCommits = gitCommits

	if len(sizes) > 0 {
		sort.Ints(sizes)
		mid := len(sizes) / 2
		if len(sizes)%2 == 1 {
			proj.MedianFunctionSize = float64(sizes[mid])
		} else {
			proj.MedianFunctionSize = float64(sizes[mid-1]+sizes[mid]) / 2.0
		}
		idxP95 := int(0.95 * float64(len(sizes)-1))
		if idxP95 < 0 {
			idxP95 = 0
		}
		if idxP95 >= len(sizes) {
			idxP95 = len(sizes) - 1
		}
		proj.P95FunctionSize = float64(sizes[idxP95])
	}

	annotateFunctionCoupling(files)
	annotateFunctionHotspots(files)

	hotspots := buildHotspots(files)

	return &model.ProjectReport{
		RootPath:       root,
		GeneratedAt:    time.Now().UTC(),
		Files:          files,
		Project:        proj,
		Hotspots:       hotspots,
		MetricMetadata: model.AllMetricSummaries(),
		Warnings:       warnings,
	}
}

func buildHotspots(files []model.FileMetrics) []model.Hotspot {
	var hs []model.Hotspot

	for _, f := range files {
		if f.Summary.CCNTotal == 0 || f.Git == nil {
			continue
		}
		churn := f.Git.LinesAdded + f.Git.LinesDeleted
		if churn == 0 {
			continue
		}
		score := float64(f.Summary.CCNTotal) * math.Log1p(float64(churn))
		hs = append(hs, model.Hotspot{
			FilePath: f.Path,
			Reason:   "complexity × churn",
			Score:    score,
			CCN:      f.Summary.CCNTotal,
			Churn:    churn,
		})
	}

	sort.Slice(hs, func(i, j int) bool {
		return hs[i].Score > hs[j].Score
	})

	if len(hs) > 10 {
		return hs[:10]
	}
	return hs
}

func annotateFunctionCoupling(files []model.FileMetrics) {
	type funcRef struct {
		fileIdx int
		fnIdx   int
	}

	byName := make(map[string][]funcRef)
	for i := range files {
		for j := range files[i].Functions {
			name := files[i].Functions[j].Name
			if name == "" {
				continue
			}
			byName[name] = append(byName[name], funcRef{fileIdx: i, fnIdx: j})
		}
	}

	for i := range files {
		for j := range files[i].Functions {
			callees := files[i].Functions[j].Callees
			for _, cname := range callees {
				refs := byName[cname]
				for _, ref := range refs {
					files[ref.fileIdx].Functions[ref.fnIdx].FanIn++
				}
			}
		}
	}
}

func annotateFunctionHotspots(files []model.FileMetrics) {
	for i := range files {
		if files[i].Git == nil {
			continue
		}
		churn := files[i].Git.LinesAdded + files[i].Git.LinesDeleted
		if churn == 0 {
			continue
		}
		factor := math.Log1p(float64(churn))
		for j := range files[i].Functions {
			fn := &files[i].Functions[j]
			fn.HotspotScore = float64(fn.CCN) * factor
		}
	}
}
