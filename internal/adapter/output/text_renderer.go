// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package output

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

const (
	ansiReset = "\033[0m"
	ansiBold  = "\033[1m"

	colMain   = "\033[38;5;223m"
	colMuted  = "\033[38;5;246m"
	colTitle  = "\033[38;5;142m"
	colAccent = "\033[38;5;208m"

	colGood   = "\033[38;5;108m"
	colWarn   = "\033[38;5;214m"
	colDanger = "\033[38;5;167m"

	colFile = "\033[38;5;67m"
	colFunc = "\033[38;5;150m"
)

type TextRenderer struct{}

func NewTextRenderer() *TextRenderer {
	return &TextRenderer{}
}

var _ ports.OutputRenderer = (*TextRenderer)(nil)

func (r *TextRenderer) Format() string {
	return "text"
}

func (r *TextRenderer) Render(report *model.ProjectReport) (string, error) {
	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", accent("CodeAudit Report"))
	fmt.Fprintf(&b, "%s %s\n", label("Root:"), value(report.RootPath))
	fmt.Fprintf(&b, "%s %s\n", label("Generated at:"), value(report.GeneratedAt.Format(time.RFC3339)))

	fmt.Fprintf(&b, "\n%s\n", title("== Project Summary =="))
	fmt.Fprintf(&b, "%s %s\n", label("Files:"), value(fmt.Sprintf("%d", report.Project.TotalFiles)))
	fmt.Fprintf(&b, "%s %s\n", label("Functions:"), value(fmt.Sprintf("%d", report.Project.TotalFunctions)))
	fmt.Fprintf(&b, "%s %s\n", label("Avg CCN / function:"), colorCCNFloat(report.Project.AvgCCNPerFunction))
	fmt.Fprintf(&b, "%s %s\n", label("Max CCN / function:"), colorCCNInt(report.Project.MaxCCNPerFunction))
	fmt.Fprintf(&b, "%s %s\n", label("Functions CCN>10:"), colorRiskPct(report.Project.FunctionsCCNGt10Pct*100))
	fmt.Fprintf(&b, "%s %s\n", label("Functions CCN>20:"), colorRiskPct(report.Project.FunctionsCCNGt20Pct*100))
	fmt.Fprintf(&b, "%s %s\n", label("Median function size:"), value(fmt.Sprintf("%.1f LOC", report.Project.MedianFunctionSize)))
	fmt.Fprintf(&b, "%s %s\n", label("P95 function size:"), value(fmt.Sprintf("%.1f LOC", report.Project.P95FunctionSize)))
	fmt.Fprintf(
		&b,
		"%s %s\n",
		label("Functions >50 / >80 / >100 LOC:"),
		value(fmt.Sprintf("%d / %d / %d",
			report.Project.FunctionsGt50Lines,
			report.Project.FunctionsGt80Lines,
			report.Project.FunctionsGt100Lines,
		)),
	)
	fmt.Fprintf(&b, "%s %s\n", label("Avg params / function:"), value(fmt.Sprintf("%.2f", report.Project.AvgParamsPerFunction)))
	fmt.Fprintf(&b, "%s %s\n", label("Comment density (avg):"), value(fmt.Sprintf("%.1f%%", report.Project.CommentDensityAvg*100)))
	fmt.Fprintf(
		&b,
		"%s %s\n",
		label("Git:"),
		value(fmt.Sprintf("commits=%d, +%d/-%d lines",
			report.Project.GitTotalCommits,
			report.Project.GitTotalLinesAdded,
			report.Project.GitTotalLinesDeleted,
		)),
	)

	if len(report.Hotspots) > 0 {
		fmt.Fprintf(&b, "\n%s\n", title("== Top Hotspots (complexity × churn) =="))
		for i, h := range report.Hotspots {
			ccnStr := colorCCNInt(h.CCN)
			scoreStr := colorHotspot(h.Score)
			fmt.Fprintf(
				&b,
				"%s %-40s %s (score=%s, CCN=%s, churn=%d)\n",
				label(fmt.Sprintf("%2d.", i+1)),
				trimPath(h.FilePath, 40),
				colMuted+"-"+ansiReset,
				scoreStr,
				ccnStr,
				h.Churn,
			)
		}
	}

	const maxFiles = 10

	files := append([]model.FileMetrics(nil), report.Files...)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Summary.CCNTotal > files[j].Summary.CCNTotal
	})

	limit := maxFiles
	if len(files) < limit {
		limit = len(files)
	}

	if limit > 0 {
		fmt.Fprintf(&b, "\n%s\n", title(fmt.Sprintf("== Files by total complexity (top %d) ==", limit)))
		for i := 0; i < limit; i++ {
			f := files[i]

			idx := fmt.Sprintf("%2d.", i+1)
			ccnRaw := fmt.Sprintf("%4d", f.Summary.CCNTotal)
			ccnField := colorCCNField(ccnRaw, f.Summary.CCNTotal)

			fmt.Fprintf(
				&b,
				"%s %-40s CCN=%s  NLOC=%5d  funcs=%3d\n",
				label(idx),
				trimPath(f.Path, 40),
				ccnField,
				f.Summary.NLOC,
				f.Summary.FunctionsCount,
			)
		}
	}

	type functionRow struct {
		File string
		Fn   model.FunctionMetrics
	}

	var rows []functionRow
	for _, f := range report.Files {
		for _, fn := range f.Functions {
			rows = append(rows, functionRow{
				File: f.Path,
				Fn:   fn,
			})
		}
	}

	if len(rows) > 0 {
		sort.Slice(rows, func(i, j int) bool {
			ci, cj := rows[i].Fn.CCN, rows[j].Fn.CCN
			if ci == cj {
				return rows[i].Fn.NLOC > rows[j].Fn.NLOC
			}
			return ci > cj
		})

		fmt.Fprintf(&b, "\n%s\n", title("== Function metrics (per function) =="))

		header := fmt.Sprintf(
			"%-40s %-30s %6s %6s %6s %6s %6s %6s %7s %7s %7s %6s %6s %8s",
			"File", "Function",
			"CCN", "COG", "NLOC",
			"Params", "Locals", "Nest",
			"LStart", "LEnd", "Cmt%%",
			"Fin", "Fout", "Hotspot",
		)
		fmt.Fprintln(&b, colMuted+header+ansiReset)
		fmt.Fprintln(&b, colMuted+strings.Repeat("-", len(header))+ansiReset)

		for _, row := range rows {
			fn := row.Fn
			cmtPct := fn.CommentDensity * 100.0

			fileRaw := fmt.Sprintf("%-40s", trimPath(row.File, 40))
			funcRaw := fmt.Sprintf("%-30s", truncate(fn.Name, 30))

			ccnRaw := fmt.Sprintf("%6d", fn.CCN)
			cogRaw := fmt.Sprintf("%6d", fn.CognitiveComplexity)
			nlocRaw := fmt.Sprintf("%6d", fn.NLOC)
			paramsRaw := fmt.Sprintf("%6d", fn.Parameters)
			localsRaw := fmt.Sprintf("%6d", fn.LocalVariables)
			nestRaw := fmt.Sprintf("%6d", fn.MaxNesting)
			lstartRaw := fmt.Sprintf("%7d", fn.StartLine)
			lendRaw := fmt.Sprintf("%7d", fn.EndLine)
			cmtRaw := fmt.Sprintf("%7.1f", cmtPct)
			finRaw := fmt.Sprintf("%6d", fn.FanIn)
			foutRaw := fmt.Sprintf("%6d", fn.FanOut)
			hotRaw := fmt.Sprintf("%8.1f", fn.HotspotScore)

			fileCol := colorFileField(fileRaw)
			funcCol := colorFuncField(funcRaw)
			ccnField := colorCCNField(ccnRaw, fn.CCN)
			cogField := colorCOGField(cogRaw, fn.CognitiveComplexity)
			hotField := colorHotspotField(hotRaw, fn.HotspotScore)

			fmt.Fprintf(
				&b,
				"%s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
				fileCol,
				funcCol,
				ccnField,
				cogField,
				nlocRaw,
				paramsRaw,
				localsRaw,
				nestRaw,
				lstartRaw,
				lendRaw,
				cmtRaw,
				finRaw,
				foutRaw,
				hotField,
			)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintf(&b, "\n%s\n", title("== Warnings =="))
		for _, w := range report.Warnings {
			fmt.Fprintf(&b, "%s %s\n", warnBullet("-"), warnText(w))
		}
	}

	return b.String(), nil
}

func title(s string) string {
	return ansiBold + colTitle + s + ansiReset
}

func accent(s string) string {
	return ansiBold + colAccent + s + ansiReset
}

func label(s string) string {
	return colMuted + s + ansiReset
}

func value(s string) string {
	return colMain + s + ansiReset
}

func warnBullet(s string) string {
	return colWarn + s + ansiReset
}

func warnText(s string) string {
	return colWarn + s + ansiReset
}

func colorFileField(s string) string {
	return colFile + s + ansiReset
}

func colorFuncField(s string) string {
	return colFunc + s + ansiReset
}

func colorCCNFloat(v float64) string {
	switch {
	case v <= 10.0:
		return colGood + fmt.Sprintf("%.2f", v) + ansiReset
	case v <= 20.0:
		return colWarn + fmt.Sprintf("%.2f", v) + ansiReset
	default:
		return colDanger + fmt.Sprintf("%.2f", v) + ansiReset
	}
}

func colorCCNInt(ccn int) string {
	switch {
	case ccn <= 10:
		return colGood + fmt.Sprintf("%d", ccn) + ansiReset
	case ccn <= 20:
		return colWarn + fmt.Sprintf("%d", ccn) + ansiReset
	default:
		return colDanger + fmt.Sprintf("%d", ccn) + ansiReset
	}
}

func colorRiskPct(p float64) string {
	switch {
	case p < 10.0:
		return colGood + fmt.Sprintf("%.1f%%", p) + ansiReset
	case p < 30.0:
		return colWarn + fmt.Sprintf("%.1f%%", p) + ansiReset
	default:
		return colDanger + fmt.Sprintf("%.1f%%", p) + ansiReset
	}
}

func colorHotspot(score float64) string {
	switch {
	case score < 20:
		return colGood + fmt.Sprintf("%.1f", score) + ansiReset
	case score < 50:
		return colWarn + fmt.Sprintf("%.1f", score) + ansiReset
	default:
		return colDanger + fmt.Sprintf("%.1f", score) + ansiReset
	}
}

func colorCCNField(raw string, ccn int) string {
	switch {
	case ccn <= 10:
		return colGood + raw + ansiReset
	case ccn <= 20:
		return colWarn + raw + ansiReset
	default:
		return colDanger + raw + ansiReset
	}
}

func colorCOGField(raw string, cog int) string {
	switch {
	case cog <= 15:
		return colGood + raw + ansiReset
	case cog <= 40:
		return colWarn + raw + ansiReset
	default:
		return colDanger + raw + ansiReset
	}
}

func colorHotspotField(raw string, score float64) string {
	switch {
	case score < 20:
		return colGood + raw + ansiReset
	case score < 50:
		return colWarn + raw + ansiReset
	default:
		return colDanger + raw + ansiReset
	}
}

func trimPath(path string, max int) string {
	if len(path) <= max {
		return path
	}
	if max <= 1 {
		return path[len(path)-max:]
	}
	return "…" + path[len(path)-max+1:]
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
