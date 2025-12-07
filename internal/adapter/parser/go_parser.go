// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"

	"github.com/rafaelvolkmer/codeaudit/internal/domain/model"
	"github.com/rafaelvolkmer/codeaudit/internal/domain/ports"
)

type GoParser struct{}

func NewGoParser() *GoParser {
	return &GoParser{}
}

var _ ports.CodeParser = (*GoParser)(nil)

func (p *GoParser) Name() string {
	return "go"
}

func (p *GoParser) SupportsFile(path string) bool {
	return strings.HasSuffix(path, ".go")
}

type lineRange struct {
	Start int
	End   int
}

func (p *GoParser) ParseFile(path string, src []byte) (*model.FileMetrics, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(src), "\n")
	totalLines := len(lines)
	commentLines := estimateCommentLines(lines)
	commentDensity := 0.0
	if totalLines > 0 {
		commentDensity = float64(commentLines) / float64(totalLines)
	}

	fm := &model.FileMetrics{
		Path:     path,
		Language: model.LanguageGo,
		Comments: model.CommentMetrics{
			TotalLines:     totalLines,
			CommentLines:   commentLines,
			CommentDensity: commentDensity,
		},
	}

	var functions []model.FunctionMetrics
	var allNloc int
	var allCcn int
	var maxCcn int
	var functionsCcnGt10, functionsCcnGt20 int
	var documentedPublic, publicCount int

	for _, decl := range file.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok || fdecl.Body == nil {
			continue
		}

		mainFn, nestedFns, pubCount, pubDocCount := analyzeGoFunction(path, lines, fset, fdecl)
		if mainFn.Name == "" {
			continue
		}

		publicCount += pubCount
		documentedPublic += pubDocCount

		allFns := append([]model.FunctionMetrics{mainFn}, nestedFns...)
		for _, fn := range allFns {
			functions = append(functions, fn)
			allNloc += fn.NLOC
			allCcn += fn.CCN
			if fn.CCN > maxCcn {
				maxCcn = fn.CCN
			}
			if fn.CCN > 10 {
				functionsCcnGt10++
			}
			if fn.CCN > 20 {
				functionsCcnGt20++
			}
		}
	}

	fm.Functions = functions
	fnCount := len(functions)
	avgCcn := 0.0
	if fnCount > 0 {
		avgCcn = float64(allCcn) / float64(fnCount)
	}
	publicDocPct := 0.0
	if publicCount > 0 {
		publicDocPct = float64(documentedPublic) / float64(publicCount)
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
	fm.Comments.PublicAPIDocPct = publicDocPct

	var smells []model.CodeSmell
	for _, fn := range functions {
		if fn.Parameters >= 5 {
			smells = append(smells, model.CodeSmell{
				Kind:        model.SmellManyParameters,
				Description: "function has many parameters (>=5)",
				FilePath:    fn.FilePath,
				Function:    fn.Name,
				Line:        fn.StartLine,
			})
		}
		if fn.LocalVariables >= 15 {
			smells = append(smells, model.CodeSmell{
				Kind:        model.SmellManyLocals,
				Description: "function has many local variables (>=15)",
				FilePath:    fn.FilePath,
				Function:    fn.Name,
				Line:        fn.StartLine,
			})
		}
		if fn.MaxNesting >= 4 {
			smells = append(smells, model.CodeSmell{
				Kind:        model.SmellDeepNesting,
				Description: "function has deep nesting (>=4)",
				FilePath:    fn.FilePath,
				Function:    fn.Name,
				Line:        fn.StartLine,
			})
		}
	}
	fm.Smells = smells

	return fm, nil
}

func analyzeGoFunction(path string, lines []string, fset *token.FileSet, fdecl *ast.FuncDecl) (model.FunctionMetrics, []model.FunctionMetrics, int, int) {
	start := fset.Position(fdecl.Pos()).Line
	end := fset.Position(fdecl.End()).Line

	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}

	funcLits := collectFuncLits(fdecl.Body)

	var excludes []lineRange
	for _, lit := range funcLits {
		s := fset.Position(lit.Pos()).Line
		e := fset.Position(lit.End()).Line
		if s < start {
			s = start
		}
		if e > end {
			e = end
		}
		if s <= e {
			excludes = append(excludes, lineRange{Start: s, End: e})
		}
	}

	nloc, ccn, cognitive, maxNesting, locals, commentLinesFn :=
		computeTextMetricsForRangeWithExcludes(lines, start, end, excludes)

	params := countParams(fdecl)
	isPublic := ast.IsExported(fdecl.Name.Name)
	isDoc := fdecl.Doc != nil && len(fdecl.Doc.List) > 0

	publicCount := 0
	documentedPublic := 0
	if isPublic {
		publicCount++
		if isDoc {
			documentedPublic++
		}
	}

	commentDensityFn := 0.0
	if nloc+commentLinesFn > 0 {
		commentDensityFn = float64(commentLinesFn) / float64(nloc+commentLinesFn)
	}

	calleeSet := make(map[string]struct{})
	ast.Inspect(fdecl.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok {
			calleeSet[ident.Name] = struct{}{}
		}
		return true
	})

	var callees []string
	for name := range calleeSet {
		callees = append(callees, name)
	}
	sort.Strings(callees)

	mainFn := model.FunctionMetrics{
		Name:                fdecl.Name.Name,
		Signature:           buildSignature(fdecl),
		FilePath:            path,
		Language:            model.LanguageGo,
		StartLine:           start,
		EndLine:             end,
		NLOC:                nloc,
		Parameters:          params,
		LocalVariables:      locals,
		CCN:                 ccn,
		CognitiveComplexity: cognitive,
		MaxNesting:          maxNesting,
		FanOut:              len(callees),
		CommentDensity:      commentDensityFn,
		Callees:             callees,
		IsPublic:            isPublic,
		IsDocumented:        isDoc,
	}

	var nestedFns []model.FunctionMetrics
	for _, lit := range funcLits {
		s := fset.Position(lit.Pos()).Line
		e := fset.Position(lit.End()).Line
		if s < 1 {
			s = 1
		}
		if e > len(lines) {
			e = len(lines)
		}
		if s > e {
			continue
		}

		nlocLit, ccnLit, cogLit, maxNestLit, localsLit, commentLinesLit :=
			computeTextMetricsForRangeWithExcludes(lines, s, e, nil)

		commentDensityLit := 0.0
		if nlocLit+commentLinesLit > 0 {
			commentDensityLit = float64(commentLinesLit) / float64(nlocLit+commentLinesLit)
		}

		paramsLit := countParamsFromFieldList(lit.Type.Params)

		calleeSetLit := make(map[string]struct{})
		ast.Inspect(lit.Body, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if ident, ok := call.Fun.(*ast.Ident); ok {
				calleeSetLit[ident.Name] = struct{}{}
			}
			return true
		})

		var calleesLit []string
		for name := range calleeSetLit {
			calleesLit = append(calleesLit, name)
		}
		sort.Strings(calleesLit)

		name := fmt.Sprintf("@%d-%d", s, e)

		nestedFns = append(nestedFns, model.FunctionMetrics{
			Name:                name,
			Signature:           name,
			FilePath:            path,
			Language:            model.LanguageGo,
			StartLine:           s,
			EndLine:             e,
			NLOC:                nlocLit,
			Parameters:          paramsLit,
			LocalVariables:      localsLit,
			CCN:                 ccnLit,
			CognitiveComplexity: cogLit,
			MaxNesting:          maxNestLit,
			FanOut:              len(calleesLit),
			CommentDensity:      commentDensityLit,
			Callees:             calleesLit,
			IsPublic:            false,
			IsDocumented:        false,
		})
	}

	return mainFn, nestedFns, publicCount, documentedPublic
}

func collectFuncLits(node ast.Node) []*ast.FuncLit {
	var lits []*ast.FuncLit
	ast.Inspect(node, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		lits = append(lits, lit)
		return true
	})
	return lits
}

func computeTextMetricsForRangeWithExcludes(lines []string, start, end int, excludes []lineRange) (nloc, ccn, cognitive, maxNesting, locals, commentLines int) {
	ccn = 1
	depth := 0
	inBlock := false

	inExcluded := func(lineNo int) bool {
		for _, r := range excludes {
			if lineNo >= r.Start && lineNo <= r.End {
				return true
			}
		}
		return false
	}

	for i := start - 1; i < end && i < len(lines); i++ {
		lineNo := i + 1
		if inExcluded(lineNo) {
			continue
		}

		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if inBlock {
			commentLines++
			if strings.Contains(trimmed, "*/") {
				inBlock = false
			}
			continue
		}

		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "//") {
			commentLines++
			continue
		}

		if idx := strings.Index(trimmed, "//"); idx >= 0 {
			codePart := strings.TrimSpace(trimmed[:idx])
			if codePart == "" {
				commentLines++
				continue
			}
			trimmed = codePart
		}

		if strings.HasPrefix(trimmed, "/*") {
			commentLines++
			if !strings.Contains(trimmed, "*/") {
				inBlock = true
			}
			continue
		}

		nloc++

		for _, ch := range line {
			switch ch {
			case '{':
				depth++
				if depth > maxNesting {
					maxNesting = depth
				}
			case '}':
				if depth > 0 {
					depth--
				}
			}
		}

		ccnLine := 0
		cogLine := 0

		if strings.Contains(trimmed, "else if ") {
			ccnLine++
			cogLine++
		} else if strings.Contains(trimmed, "if ") {
			ccnLine++
			cogLine++
		}

		if strings.Contains(trimmed, "for ") {
			ccnLine++
			cogLine++
		}
		if strings.Contains(trimmed, "switch ") {
			ccnLine++
			cogLine++
		}

		caseCount := strings.Count(trimmed, "case ")
		if caseCount > 0 {
			ccnLine += caseCount
			cogLine += caseCount
		}
		if strings.Contains(trimmed, "default:") {
			ccnLine++
			cogLine++
		}
		if strings.Contains(trimmed, "goto ") {
			ccnLine++
			cogLine++
		}

		boolOps := strings.Count(trimmed, "&&") + strings.Count(trimmed, "||")
		if boolOps > 0 {
			cogLine += boolOps
		}

		if strings.HasPrefix(trimmed, "return ") && depth > 0 {
			cogLine++
		}

		if ccnLine > 0 {
			ccn += ccnLine
		}
		if cogLine > 0 {
			cognitive += cogLine * (1 + depth)
		}

		if strings.Contains(line, ":=") || strings.HasPrefix(trimmed, "var ") {
			locals++
		}
	}

	return nloc, ccn, cognitive, maxNesting, locals, commentLines
}

func countParams(fn *ast.FuncDecl) int {
	if fn.Type == nil || fn.Type.Params == nil {
		return 0
	}
	return countParamsFromFieldList(fn.Type.Params)
}

func countParamsFromFieldList(fl *ast.FieldList) int {
	if fl == nil {
		return 0
	}
	total := 0
	for _, f := range fl.List {
		if len(f.Names) == 0 {
			total++
		} else {
			total += len(f.Names)
		}
	}
	return total
}

func buildSignature(fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	return fn.Name.Name
}
