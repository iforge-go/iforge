package service

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	gitsvc "iforge/iforge/internal/git"
	"iforge/iforge/internal/model"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gorm.io/gorm"
)

// CodeQualityService handles code quality analysis
type CodeQualityService struct {
	db        *gorm.DB
	gitClient *gitsvc.Client
}

// NewCodeQualityService creates a new CodeQualityService
func NewCodeQualityService(db *gorm.DB, gitClient *gitsvc.Client) *CodeQualityService {
	return &CodeQualityService{db: db, gitClient: gitClient}
}

// CodeQualityReport represents a code quality analysis report
type CodeQualityReport struct {
	TotalLines      int            `json:"totalLines"`
	TotalFiles      int            `json:"totalFiles"`
	LanguageStats   map[string]int `json:"languageStats"`
	AvgLineLength   float64        `json:"avgLineLength"`
	MaxLineLength   int            `json:"maxLineLength"`
	CommentRatio    float64        `json:"commentRatio"`
	EmptyLineRatio  float64        `json:"emptyLineRatio"`
	LongFunctions   []FunctionInfo `json:"longFunctions"`
	LargeFiles      []FileInfo     `json:"largeFiles"`
	ComplexityScore float64        `json:"complexityScore"`
	Maintainability string         `json:"maintainability"`
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name   string `json:"name"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Length int    `json:"length"`
}

// FileInfo represents information about a file
type FileInfo struct {
	Path  string `json:"path"`
	Lines int    `json:"lines"`
	Size  int64  `json:"size"`
}

// AnalyzeCodeQuality analyzes the code quality of a repository
func (s *CodeQualityService) AnalyzeCodeQuality(owner, repo string) (*CodeQualityReport, error) {
	// Get the default branch
	var repoModel model.Repository
	err := s.db.
		Where("user_name = ? AND repository_name = ?", owner, repo).
		First(&repoModel).Error
	if err != nil {
		return nil, err
	}

	// Open the bare git repository
	r, err := s.gitClient.OpenRepository(owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to open repository: %w", err)
	}

	// Resolve the branch
	branch := repoModel.DefaultBranch
	if branch == "" {
		branch = "master"
	}
	refName := plumbing.ReferenceName("refs/heads/" + branch)
	headRef, err := r.Reference(refName, true)
	if err != nil {
		headRef, err = r.Head()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve reference: %w", err)
		}
	}

	commit, err := r.CommitObject(headRef.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to get tree: %w", err)
	}

	// Initialize report
	report := &CodeQualityReport{
		LanguageStats: make(map[string]int),
	}

	// Accumulators for aggregate metrics
	var totalComments, totalEmptyLines, totalLineLength int

	// Walk through all files in the tree recursively
	files := tree.Files()
	defer files.Close()

	err = files.ForEach(func(f *object.File) error {
		ext := strings.ToLower(filepath.Ext(f.Name))
		if isBinaryFile(ext) {
			return nil
		}
		language := detectLanguage(ext)
		if language == "" {
			return nil
		}

		content, err := f.Contents()
		if err != nil {
			return nil
		}

		lineCount := 0
		commentCount := 0
		emptyLineCount := 0
		fileLineLength := 0
		maxLineLength := 0

		scanner := bufio.NewScanner(strings.NewReader(content))
		scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			lineCount++
			lineLength := len(line)
			fileLineLength += lineLength
			if lineLength > maxLineLength {
				maxLineLength = lineLength
			}
			if strings.TrimSpace(line) == "" {
				emptyLineCount++
				continue
			}
			if isCommentLine(line, language) {
				commentCount++
			}
		}

		report.TotalLines += lineCount
		report.TotalFiles++
		report.LanguageStats[language] += lineCount
		totalComments += commentCount
		totalEmptyLines += emptyLineCount
		totalLineLength += fileLineLength

		if maxLineLength > report.MaxLineLength {
			report.MaxLineLength = maxLineLength
		}

		if lineCount > 500 {
			report.LargeFiles = append(report.LargeFiles, FileInfo{
				Path:  f.Name,
				Lines: lineCount,
				Size:  f.Size,
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Calculate aggregate ratios
	if report.TotalLines > 0 {
		report.CommentRatio = float64(totalComments) / float64(report.TotalLines)
		report.EmptyLineRatio = float64(totalEmptyLines) / float64(report.TotalLines)
		report.AvgLineLength = float64(totalLineLength) / float64(report.TotalLines)
		report.ComplexityScore = s.calculateComplexity(report)
		report.Maintainability = s.calculateMaintainability(report.ComplexityScore)
	}

	return report, nil
}

// calculateComplexity calculates a complexity score
func (s *CodeQualityService) calculateComplexity(report *CodeQualityReport) float64 {
	score := 0.0
	if report.AvgLineLength > 80 {
		score += (report.AvgLineLength - 80) * 0.1
	}
	if report.CommentRatio < 0.2 {
		score += (0.2 - report.CommentRatio) * 100
	}
	score += float64(len(report.LargeFiles)) * 10
	score += float64(len(report.LongFunctions)) * 5
	return score
}

// calculateMaintainability calculates maintainability grade
func (s *CodeQualityService) calculateMaintainability(score float64) string {
	if score < 10 {
		return "A"
	} else if score < 20 {
		return "B"
	} else if score < 30 {
		return "C"
	} else if score < 50 {
		return "D"
	}
	return "F"
}

// isBinaryFile checks if a file is binary based on extension
func isBinaryFile(ext string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".obj": true, ".o": true, ".a": true, ".lib": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".svg": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wav": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
		".pyc": true, ".class": true,
	}
	return binaryExts[ext]
}

// detectLanguage detects the programming language based on file extension
func detectLanguage(ext string) string {
	languageMap := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".jsx":   "JavaScript",
		".ts":    "TypeScript",
		".tsx":   "TypeScript",
		".py":    "Python",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".cc":    "C++",
		".h":     "C/C++ Header",
		".hpp":   "C++ Header",
		".cs":    "C#",
		".php":   "PHP",
		".rb":    "Ruby",
		".rs":    "Rust",
		".swift": "Swift",
		".kt":    "Kotlin",
		".scala": "Scala",
		".sh":    "Shell",
		".bash":  "Shell",
		".zsh":   "Shell",
		".html":  "HTML",
		".htm":   "HTML",
		".css":   "CSS",
		".scss":  "SCSS",
		".sass":  "Sass",
		".less":  "Less",
		".json":  "JSON",
		".xml":   "XML",
		".yaml":  "YAML",
		".yml":   "YAML",
		".md":    "Markdown",
		".sql":   "SQL",
		".r":     "R",
		".pl":    "Perl",
		".lua":   "Lua",
	}
	return languageMap[ext]
}

// isCommentLine checks if a line is a comment
func isCommentLine(line string, language string) bool {
	line = strings.TrimSpace(line)
	commentPatterns := map[string][]string{
		"Go":           {"//", "/*", "*"},
		"JavaScript":   {"//", "/*", "*"},
		"TypeScript":   {"//", "/*", "*"},
		"Python":       {"#"},
		"Java":         {"//", "/*", "*"},
		"C":            {"//", "/*", "*"},
		"C++":          {"//", "/*", "*"},
		"C/C++ Header": {"//", "/*", "*"},
		"C++ Header":   {"//", "/*", "*"},
		"C#":           {"//", "/*", "*"},
		"PHP":          {"//", "/*", "*", "#"},
		"Ruby":         {"#"},
		"Rust":         {"//", "/*", "*"},
		"Swift":        {"//", "/*", "*"},
		"Kotlin":       {"//", "/*", "*"},
		"Scala":        {"//", "/*", "*"},
		"Shell":        {"#"},
		"Perl":         {"#"},
		"Lua":          {"--"},
		"SQL":          {"--"},
		"R":            {"#"},
	}
	patterns, ok := commentPatterns[language]
	if !ok {
		return false
	}
	for _, pattern := range patterns {
		if strings.HasPrefix(line, pattern) {
			return true
		}
	}
	return false
}
