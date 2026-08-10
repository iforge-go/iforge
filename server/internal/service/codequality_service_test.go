package service

import (
	"testing"
)

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".exe", true},
		{".dll", true},
		{".so", true},
		{".dylib", true},
		{".bin", true},
		{".obj", true},
		{".o", true},
		{".a", true},
		{".lib", true},
		{".zip", true},
		{".tar", true},
		{".gz", true},
		{".rar", true},
		{".7z", true},
		{".png", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".bmp", true},
		{".ico", true},
		{".svg", true},
		{".mp3", true},
		{".mp4", true},
		{".avi", true},
		{".mov", true},
		{".wav", true},
		{".pdf", true},
		{".doc", true},
		{".docx", true},
		{".xls", true},
		{".xlsx", true},
		{".pyc", true},
		{".class", true},
		{".go", false},
		{".js", false},
		{".ts", false},
		{".py", false},
		{".java", false},
		{".c", false},
		{".cpp", false},
		{".h", false},
		{".cs", false},
		{".php", false},
		{".rb", false},
		{".rs", false},
		{".swift", false},
		{".kt", false},
		{".scala", false},
		{".sh", false},
		{".bash", false},
		{".zsh", false},
		{".html", false},
		{".htm", false},
		{".css", false},
		{".scss", false},
		{".sass", false},
		{".less", false},
		{".json", false},
		{".xml", false},
		{".yaml", false},
		{".yml", false},
		{".md", false},
		{".sql", false},
		{".r", false},
		{".pl", false},
		{".lua", false},
		{".txt", false},
		{".log", false},
		{".csv", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := isBinaryFile(tt.ext)
			if result != tt.expected {
				t.Errorf("isBinaryFile(%q) = %v; want %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "Go"},
		{".js", "JavaScript"},
		{".jsx", "JavaScript"},
		{".ts", "TypeScript"},
		{".tsx", "TypeScript"},
		{".py", "Python"},
		{".java", "Java"},
		{".c", "C"},
		{".cpp", "C++"},
		{".cc", "C++"},
		{".h", "C/C++ Header"},
		{".hpp", "C++ Header"},
		{".cs", "C#"},
		{".php", "PHP"},
		{".rb", "Ruby"},
		{".rs", "Rust"},
		{".swift", "Swift"},
		{".kt", "Kotlin"},
		{".scala", "Scala"},
		{".sh", "Shell"},
		{".bash", "Shell"},
		{".zsh", "Shell"},
		{".html", "HTML"},
		{".htm", "HTML"},
		{".css", "CSS"},
		{".scss", "SCSS"},
		{".sass", "Sass"},
		{".less", "Less"},
		{".json", "JSON"},
		{".xml", "XML"},
		{".yaml", "YAML"},
		{".yml", "YAML"},
		{".md", "Markdown"},
		{".sql", "SQL"},
		{".r", "R"},
		{".pl", "Perl"},
		{".lua", "Lua"},
		{".txt", ""},
		{".log", ""},
		{".csv", ""},
		{".unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := detectLanguage(tt.ext)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q; want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestIsCommentLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		language string
		expected bool
	}{
		// Go comments
		{"Go single line comment", "// This is a comment", "Go", true},
		{"Go block comment start", "/* This is a block comment", "Go", true},
		{"Go block comment middle", "* This is a block comment", "Go", true},
		{"Go non-comment", "func main() {", "Go", false},
		{"Go comment with leading space", "  // Comment with space", "Go", true},

		// JavaScript comments
		{"JS single line comment", "// JS comment", "JavaScript", true},
		{"JS block comment", "/* JS block", "JavaScript", true},
		{"JS non-comment", "const x = 1;", "JavaScript", false},

		// TypeScript comments
		{"TS single line comment", "// TS comment", "TypeScript", true},
		{"TS non-comment", "let x: number = 1;", "TypeScript", false},

		// Python comments
		{"Python comment", "# Python comment", "Python", true},
		{"Python non-comment", "def main():", "Python", false},

		// Java comments
		{"Java single line", "// Java comment", "Java", true},
		{"Java block", "/* Java block", "Java", true},
		{"Java non-comment", "public class Main {", "Java", false},

		// C comments
		{"C single line", "// C comment", "C", true},
		{"C block", "/* C block", "C", true},
		{"C non-comment", "int main() {", "C", false},

		// C++ comments
		{"C++ single line", "// C++ comment", "C++", true},
		{"C++ non-comment", "class MyClass {", "C++", false},

		// C# comments
		{"C# single line", "// C# comment", "C#", true},
		{"C# non-comment", "namespace MyApp {", "C#", false},

		// PHP comments
		{"PHP single line //", "// PHP comment", "PHP", true},
		{"PHP single line #", "# PHP comment", "PHP", true},
		{"PHP block", "/* PHP block", "PHP", true},
		{"PHP non-comment", "<?php echo 'hello';", "PHP", false},

		// Ruby comments
		{"Ruby comment", "# Ruby comment", "Ruby", true},
		{"Ruby non-comment", "def hello", "Ruby", false},

		// Rust comments
		{"Rust single line", "// Rust comment", "Rust", true},
		{"Rust block", "/* Rust block", "Rust", true},
		{"Rust non-comment", "fn main() {", "Rust", false},

		// Swift comments
		{"Swift single line", "// Swift comment", "Swift", true},
		{"Swift non-comment", "var x = 1", "Swift", false},

		// Kotlin comments
		{"Kotlin single line", "// Kotlin comment", "Kotlin", true},
		{"Kotlin non-comment", "fun main() {", "Kotlin", false},

		// Scala comments
		{"Scala single line", "// Scala comment", "Scala", true},
		{"Scala non-comment", "object Main {", "Scala", false},

		// Shell comments
		{"Shell comment", "# Shell comment", "Shell", true},
		{"Shell non-comment", "echo 'hello'", "Shell", false},

		// Perl comments
		{"Perl comment", "# Perl comment", "Perl", true},
		{"Perl non-comment", "print 'hello';", "Perl", false},

		// Lua comments
		{"Lua comment", "-- Lua comment", "Lua", true},
		{"Lua non-comment", "print('hello')", "Lua", false},

		// SQL comments
		{"SQL comment", "-- SQL comment", "SQL", true},
		{"SQL non-comment", "SELECT * FROM users;", "SQL", false},

		// R comments
		{"R comment", "# R comment", "R", true},
		{"R non-comment", "x <- 1", "R", false},

		// Unknown language
		{"Unknown language", "// Some comment", "Unknown", false},

		// Empty lines
		{"Empty line", "", "Go", false},
		{"Whitespace only", "   ", "Go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommentLine(tt.line, tt.language)
			if result != tt.expected {
				t.Errorf("isCommentLine(%q, %q) = %v; want %v", tt.line, tt.language, result, tt.expected)
			}
		})
	}
}

func TestCalculateComplexity(t *testing.T) {
	service := &CodeQualityService{}

	tests := []struct {
		name     string
		report   *CodeQualityReport
		expected float64
	}{
		{
			name: "Low complexity - good code",
			report: &CodeQualityReport{
				AvgLineLength: 60,
				CommentRatio:  0.3,
				LargeFiles:    []FileInfo{},
				LongFunctions: []FunctionInfo{},
			},
			expected: 0,
		},
		{
			name: "Medium complexity - long lines",
			report: &CodeQualityReport{
				AvgLineLength: 100,
				CommentRatio:  0.3,
				LargeFiles:    []FileInfo{},
				LongFunctions: []FunctionInfo{},
			},
			expected: 2.0, // (100-80) * 0.1 = 2.0
		},
		{
			name: "Medium complexity - low comment ratio",
			report: &CodeQualityReport{
				AvgLineLength: 60,
				CommentRatio:  0.1,
				LargeFiles:    []FileInfo{},
				LongFunctions: []FunctionInfo{},
			},
			expected: 10.0, // (0.2-0.1) * 100 = 10.0
		},
		{
			name: "High complexity - large files",
			report: &CodeQualityReport{
				AvgLineLength: 60,
				CommentRatio:  0.3,
				LargeFiles:    []FileInfo{{Path: "large.go", Lines: 600}, {Path: "big.go", Lines: 700}},
				LongFunctions: []FunctionInfo{},
			},
			expected: 20.0, // 2 * 10 = 20.0
		},
		{
			name: "High complexity - long functions",
			report: &CodeQualityReport{
				AvgLineLength: 60,
				CommentRatio:  0.3,
				LargeFiles:    []FileInfo{},
				LongFunctions: []FunctionInfo{{Name: "longFunc", Length: 100}, {Name: "hugeFunc", Length: 150}},
			},
			expected: 10.0, // 2 * 5 = 10.0
		},
		{
			name: "Very high complexity - all factors",
			report: &CodeQualityReport{
				AvgLineLength: 100,
				CommentRatio:  0.1,
				LargeFiles:    []FileInfo{{Path: "large.go", Lines: 600}},
				LongFunctions: []FunctionInfo{{Name: "longFunc", Length: 100}},
			},
			expected: 27.0, // 2.0 + 10.0 + 10.0 + 5.0 = 27.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateComplexity(tt.report)
			if result != tt.expected {
				t.Errorf("calculateComplexity() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateMaintainability(t *testing.T) {
	service := &CodeQualityService{}

	tests := []struct {
		name     string
		score    float64
		expected string
	}{
		{"Grade A - excellent", 5.0, "A"},
		{"Grade A - threshold", 9.99, "A"},
		{"Grade B - low", 10.0, "B"},
		{"Grade B - mid", 15.0, "B"},
		{"Grade B - threshold", 19.99, "B"},
		{"Grade C - low", 20.0, "C"},
		{"Grade C - mid", 25.0, "C"},
		{"Grade C - threshold", 29.99, "C"},
		{"Grade D - low", 30.0, "D"},
		{"Grade D - mid", 40.0, "D"},
		{"Grade D - threshold", 49.99, "D"},
		{"Grade F - low", 50.0, "F"},
		{"Grade F - high", 100.0, "F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateMaintainability(tt.score)
			if result != tt.expected {
				t.Errorf("calculateMaintainability(%v) = %q; want %q", tt.score, result, tt.expected)
			}
		})
	}
}
