// SPDX-FileCopyrightText: 2024-2025 Rafael V. Volkmer <rafael.v.volkmer@gmail.com>
// SPDX-License-Identifier: MIT

package model

import "time"

type Language string

const (
	LanguageUnknown Language = "unknown"
	LanguageGo      Language = "go"
	LanguageC       Language = "c"
	LanguageCpp     Language = "cpp"
)

type MetricID string

const (
	MetricCyclomaticCCN        MetricID = "complexity.ccn"
	MetricCognitiveComplexity  MetricID = "complexity.cognitive"
	MetricMaxNesting           MetricID = "complexity.max_nesting"
	MetricNLOC                 MetricID = "size.nloc"
	MetricFunctionNLOC         MetricID = "size.function_nloc"
	MetricParamsCount          MetricID = "params.count"
	MetricLocalsCount          MetricID = "locals.count"
	MetricFanIn                MetricID = "coupling.fan_in"
	MetricFanOut               MetricID = "coupling.fan_out"
	MetricAfferentCoupling     MetricID = "coupling.afferent"
	MetricEfferentCoupling     MetricID = "coupling.efferent"
	MetricInstability          MetricID = "coupling.instability"
	MetricCommentDensity       MetricID = "comments.density"
	MetricPublicAPIDocCoverage MetricID = "comments.public_api_doc"
	MetricCloneDensity         MetricID = "clones.density"
	MetricSmellsCount          MetricID = "smells.count"
	MetricGitLinesAdded        MetricID = "git.churn.lines_added"
	MetricGitLinesDeleted      MetricID = "git.churn.lines_deleted"
	MetricGitCommits           MetricID = "git.commits"
	MetricGitBugfixCommits     MetricID = "git.commits.bugfix"
	MetricGitAuthors           MetricID = "git.authors"
	MetricHotspotScore         MetricID = "hotspot.score_complexity_churn"
)

type FunctionMetrics struct {
	Name                string   `json:"name"`
	Signature           string   `json:"signature"`
	FilePath            string   `json:"filePath"`
	Language            Language `json:"language"`
	StartLine           int      `json:"startLine"`
	EndLine             int      `json:"endLine"`
	NLOC                int      `json:"nloc"`
	Parameters          int      `json:"parameters"`
	LocalVariables      int      `json:"localVariables"`
	CCN                 int      `json:"ccn"`
	CognitiveComplexity int      `json:"cognitiveComplexity"`
	MaxNesting          int      `json:"maxNesting"`
	FanIn               int      `json:"fanIn"`
	FanOut              int      `json:"fanOut"`
	CommentDensity      float64  `json:"commentDensity"`
	HotspotScore        float64  `json:"hotspotScore,omitempty"`
	Callees             []string `json:"callees,omitempty"`
	IsPublic            bool     `json:"isPublic"`
	IsDocumented        bool     `json:"isDocumented"`
}

type CommentMetrics struct {
	TotalLines      int     `json:"totalLines"`
	CommentLines    int     `json:"commentLines"`
	CommentDensity  float64 `json:"commentDensity"`
	PublicAPIDocPct float64 `json:"publicApiDocPct"`
}

type CodeSmellKind string

const (
	SmellManyParameters CodeSmellKind = "many_parameters"
	SmellManyLocals     CodeSmellKind = "many_locals"
	SmellDeepNesting    CodeSmellKind = "deep_nesting"
	SmellGodFunction    CodeSmellKind = "god_function"
	SmellGlobalState    CodeSmellKind = "global_state"
)

type CodeSmell struct {
	Kind        CodeSmellKind `json:"kind"`
	Description string        `json:"description"`
	FilePath    string        `json:"filePath"`
	Function    string        `json:"function,omitempty"`
	Line        int           `json:"line,omitempty"`
}

type GitFileMetrics struct {
	FilePath      string `json:"filePath"`
	LinesAdded    int    `json:"linesAdded"`
	LinesDeleted  int    `json:"linesDeleted"`
	Commits       int    `json:"commits"`
	BugfixCommits int    `json:"bugfixCommits"`
	Authors       int    `json:"authors"`
}

type FileSummaryMetrics struct {
	NLOC              int     `json:"nloc"`
	CCNTotal          int     `json:"ccnTotal"`
	CCNAvgPerFunction float64 `json:"ccnAvgPerFunction"`
	CCNMaxFunction    int     `json:"ccnMaxFunction"`
	FunctionsCount    int     `json:"functionsCount"`
	FunctionsCCNGt10  int     `json:"functionsCcnGt10"`
	FunctionsCCNGt20  int     `json:"functionsCcnGt20"`
}

type FileMetrics struct {
	Path      string             `json:"path"`
	Language  Language           `json:"language"`
	Summary   FileSummaryMetrics `json:"summary"`
	Functions []FunctionMetrics  `json:"functions"`
	Comments  CommentMetrics     `json:"comments"`
	Smells    []CodeSmell        `json:"smells"`
	Git       *GitFileMetrics    `json:"git,omitempty"`
}

type Hotspot struct {
	FilePath string  `json:"filePath"`
	Reason   string  `json:"reason"`
	Score    float64 `json:"score"`
	CCN      int     `json:"ccn"`
	Churn    int     `json:"churn"`
}

type ProjectMetrics struct {
	TotalFiles          int     `json:"totalFiles"`
	TotalFunctions      int     `json:"totalFunctions"`
	AvgCCNPerFunction   float64 `json:"avgCcnPerFunction"`
	MaxCCNPerFunction   int     `json:"maxCcnPerFunction"`
	FunctionsCCNGt10Pct float64 `json:"functionsCcnGt10Pct"`
	FunctionsCCNGt20Pct float64 `json:"functionsCcnGt20Pct"`

	MedianFunctionSize  float64 `json:"medianFunctionSize"`
	P95FunctionSize     float64 `json:"p95FunctionSize"`
	FunctionsGt50Lines  int     `json:"functionsGt50Lines"`
	FunctionsGt80Lines  int     `json:"functionsGt80Lines"`
	FunctionsGt100Lines int     `json:"functionsGt100Lines"`

	AvgParamsPerFunction float64 `json:"avgParamsPerFunction"`
	FunctionsParamsGe5   int     `json:"functionsParamsGe5"`

	CommentDensityAvg float64 `json:"commentDensityAvg"`

	GitTotalLinesAdded   int `json:"gitTotalLinesAdded"`
	GitTotalLinesDeleted int `json:"gitTotalLinesDeleted"`
	GitTotalCommits      int `json:"gitTotalCommits"`
}

type MetricSummary struct {
	ID          MetricID `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Group       string   `json:"group"`
}

type ProjectReport struct {
	RootPath       string          `json:"rootPath"`
	GeneratedAt    time.Time       `json:"generatedAt"`
	Files          []FileMetrics   `json:"files"`
	Project        ProjectMetrics  `json:"project"`
	Hotspots       []Hotspot       `json:"hotspots"`
	MetricMetadata []MetricSummary `json:"metricMetadata"`
	Warnings       []string        `json:"warnings,omitempty"`
}

func AllMetricSummaries() []MetricSummary {
	return []MetricSummary{
		{
			ID:          MetricCyclomaticCCN,
			Name:        "Cyclomatic Complexity (CCN)",
			Description: "Branching-based complexity per function/file/module.",
			Group:       "complexity",
		},
		{
			ID:          MetricCognitiveComplexity,
			Name:        "Cognitive Complexity",
			Description: "Nesting and boolean-logic–aware complexity per function.",
			Group:       "complexity",
		},
		{
			ID:          MetricMaxNesting,
			Name:        "Max Nesting Depth",
			Description: "Maximum depth of nested control structures.",
			Group:       "complexity",
		},
		{
			ID:          MetricNLOC,
			Name:        "NLOC",
			Description: "Non-empty, non-comment logical lines of code per file.",
			Group:       "size",
		},
		{
			ID:          MetricFunctionNLOC,
			Name:        "Function NLOC",
			Description: "Lines of code per function for distribution analysis.",
			Group:       "size",
		},
		{
			ID:          MetricParamsCount,
			Name:        "Parameter Count",
			Description: "Number of parameters per function.",
			Group:       "size",
		},
		{
			ID:          MetricLocalsCount,
			Name:        "Local Variables Count",
			Description: "Number of local variables per function.",
			Group:       "size",
		},
		{
			ID:          MetricFanIn,
			Name:        "Fan-in",
			Description: "How many functions depend on a given function (callers).",
			Group:       "coupling",
		},
		{
			ID:          MetricFanOut,
			Name:        "Fan-out",
			Description: "How many functions a given function depends on (callees).",
			Group:       "coupling",
		},
		{
			ID:          MetricAfferentCoupling,
			Name:        "Afferent Coupling (Ca)",
			Description: "Number of modules that depend on this module.",
			Group:       "coupling",
		},
		{
			ID:          MetricEfferentCoupling,
			Name:        "Efferent Coupling (Ce)",
			Description: "Number of modules this module depends on.",
			Group:       "coupling",
		},
		{
			ID:          MetricInstability,
			Name:        "Instability",
			Description: "Ce / (Ca + Ce), 0 = stable, 1 = unstable.",
			Group:       "coupling",
		},
		{
			ID:          MetricCommentDensity,
			Name:        "Comment Density",
			Description: "Ratio of comment lines to total lines.",
			Group:       "comments",
		},
		{
			ID:          MetricPublicAPIDocCoverage,
			Name:        "Public API Doc Coverage",
			Description: "Percentage of public functions with documentation.",
			Group:       "comments",
		},
		{
			ID:          MetricCloneDensity,
			Name:        "Clone Density",
			Description: "Estimated amount of duplicated code.",
			Group:       "clones",
		},
		{
			ID:          MetricSmellsCount,
			Name:        "Code Smells",
			Description: "Count of simple structural smells (many params, deep nesting, etc.).",
			Group:       "smells",
		},
		{
			ID:          MetricGitLinesAdded,
			Name:        "Git Lines Added",
			Description: "Lines added in Git history for a file.",
			Group:       "git",
		},
		{
			ID:          MetricGitLinesDeleted,
			Name:        "Git Lines Deleted",
			Description: "Lines deleted in Git history for a file.",
			Group:       "git",
		},
		{
			ID:          MetricGitCommits,
			Name:        "Git Commits",
			Description: "Number of commits touching a file.",
			Group:       "git",
		},
		{
			ID:          MetricGitBugfixCommits,
			Name:        "Bugfix Commits",
			Description: "Number of commits that look like bug fixes.",
			Group:       "git",
		},
		{
			ID:          MetricGitAuthors,
			Name:        "Authors",
			Description: "Number of distinct authors touching a file (bus factor proxy).",
			Group:       "git",
		},
		{
			ID:          MetricHotspotScore,
			Name:        "Hotspot Score",
			Description: "Heuristic score combining complexity and churn.",
			Group:       "hotspots",
		},
	}
}
