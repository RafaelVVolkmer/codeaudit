// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package parser

import (
	"regexp"
	"sort"
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type CParser struct {
	funcHeaderRe *regexp.Regexp
}

func NewCParser() *CParser {
	return &CParser{
		funcHeaderRe: regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\([^()]*\)\s*$`),
	}
}

var _ ports.CodeParser = (*CParser)(nil)

func (p *CParser) Name() string {
	return "c/c++"
}

func (p *CParser) SupportsFile(path string) bool {
	for _, ext := range []string{".c", ".h", ".cpp", ".hpp", ".cc", ".hh"} {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func (p *CParser) ParseFile(path string, src []byte) (*model.FileMetrics, error) {
	text := string(src)
	lines := strings.Split(text, "\n")

	totalLines := len(lines)
	commentLines := estimateCommentLines(lines)
	density := 0.0
	if totalLines > 0 {
		density = float64(commentLines) / float64(totalLines)
	}

	fm := &model.FileMetrics{
		Path:     path,
		Language: model.LanguageC,
		Comments: model.CommentMetrics{
			TotalLines:     totalLines,
			CommentLines:   commentLines,
			CommentDensity: density,
		},
	}

	var functions []model.FunctionMetrics
	var allNloc, allCcn, maxCcn int
	var functionsCcnGt10, functionsCcnGt20 int

	inFunc := false
	funcStart := 0
	funcName := ""
	braceDepth := 0

	var headerBuf strings.Builder
	headerStart := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inFunc {

			if trimmed == "" ||
				strings.HasPrefix(trimmed, "//") ||
				strings.HasPrefix(trimmed, "/*") ||
				strings.HasPrefix(trimmed, "*") ||
				strings.HasPrefix(trimmed, "#") {
				headerBuf.Reset()
				headerStart = -1
				continue
			}

			if headerStart == -1 {
				headerStart = i + 1
			}
			if headerBuf.Len() > 0 {
				headerBuf.WriteByte(' ')
			}
			headerBuf.WriteString(trimmed)

			if strings.Contains(trimmed, "{") {
				candidate := headerBuf.String()
				if idx := strings.Index(candidate, "{"); idx >= 0 {
					candidate = strings.TrimSpace(candidate[:idx])
				}

				if m := p.funcHeaderRe.FindStringSubmatch(candidate); len(m) == 2 {
					name := m[1]
					if !isControlKeyword(name) {
						inFunc = true
						funcName = name
						funcStart = headerStart

						headerText := strings.Join(lines[funcStart-1:i+1], "\n")
						braceDepth = strings.Count(headerText, "{") - strings.Count(headerText, "}")
					}
				}

				headerBuf.Reset()
				headerStart = -1
			}

			continue
		}

		braceDepth += strings.Count(line, "{")
		braceDepth -= strings.Count(line, "}")

		if braceDepth <= 0 {
			start := funcStart
			end := i + 1

			nloc, ccn, cognitive, maxNesting, locals, commentLinesFn :=
				computeTextMetricsForRange(lines, start, end)

			commentDensityFn := 0.0
			if nloc+commentLinesFn > 0 {
				commentDensityFn = float64(commentLinesFn) / float64(nloc+commentLinesFn)
			}

			callees := extractCFunctionCalls(lines, start, end)

			fn := model.FunctionMetrics{
				Name:                funcName,
				Signature:           funcName,
				FilePath:            path,
				Language:            model.LanguageC,
				StartLine:           start,
				EndLine:             end,
				NLOC:                nloc,
				CCN:                 ccn,
				CognitiveComplexity: cognitive,
				MaxNesting:          maxNesting,
				LocalVariables:      locals,
				FanOut:              len(callees),
				CommentDensity:      commentDensityFn,
				Callees:             callees,
			}

			functions = append(functions, fn)
			allNloc += nloc
			allCcn += ccn
			if ccn > maxCcn {
				maxCcn = ccn
			}
			if ccn > 10 {
				functionsCcnGt10++
			}
			if ccn > 20 {
				functionsCcnGt20++
			}

			inFunc = false
			funcName = ""
			funcStart = 0
			braceDepth = 0
		}
	}

	fm.Functions = functions
	fnCount := len(functions)
	avgCcn := 0.0
	if fnCount > 0 {
		avgCcn = float64(allCcn) / float64(fnCount)
	}

	fm.Summary = model.FileSummaryMetrics{
		NLOC:              allNloc,
		CCNTotal:          allCcn,
		CCNAvgPerFunction: avgCcn,
		CCNMaxFunction:    maxCcn,
		FunctionsCount:    fnCount,
		FunctionsCCNGt10:  functionsCcnGt10,
		FunctionsCCNGt20:  functionsCcnGt20,
	}

	return fm, nil
}

var cCallRegexp = regexp.MustCompile(`\b([a-zA-Z_]\w*)\s*\(`)

func extractCFunctionCalls(lines []string, start, end int) []string {
	seen := make(map[string]struct{})

	for i := start - 1; i < end && i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		matches := cCallRegexp.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) < 2 {
				continue
			}
			name := m[1]
			if isControlKeyword(name) || name == "sizeof" {
				continue
			}
			seen[name] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func isControlKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "return":
		return true
	default:
		return false
	}
}
