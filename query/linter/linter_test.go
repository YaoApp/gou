package linter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// getTestAssetsDir returns the absolute path to the test assets directory
func getTestAssetsDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current file path")
	}
	// linter_test.go is in query/linter/, assets are in query/assets/linter/
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "assets", "linter")
}

// TestCase represents a single test case from testcases.json
type TestCase struct {
	File             string   `json:"file"`
	Description      string   `json:"description"`
	ExpectedErrors   []string `json:"expected_errors,omitempty"`
	ExpectedCount    int      `json:"expected_count,omitempty"`
	ExpectedCountMin int      `json:"expected_count_min,omitempty"`
}

// TestCases represents the structure of testcases.json
type TestCases struct {
	Description string `json:"description"`
	Valid       struct {
		Description string     `json:"description"`
		Cases       []TestCase `json:"cases"`
	} `json:"valid"`
	Invalid struct {
		Description string     `json:"description"`
		Cases       []TestCase `json:"cases"`
	} `json:"invalid"`
}

func loadTestCases(t *testing.T) *TestCases {
	t.Helper()
	assetsDir := getTestAssetsDir()
	data, err := os.ReadFile(filepath.Join(assetsDir, "testcases.json"))
	if err != nil {
		t.Fatalf("Failed to load testcases.json: %v", err)
	}

	var cases TestCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("Failed to parse testcases.json: %v", err)
	}
	return &cases
}

func loadTestFile(t *testing.T, filename string) string {
	t.Helper()
	assetsDir := getTestAssetsDir()
	path := filepath.Join(assetsDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to load test file %s: %v", filename, err)
	}
	return string(data)
}

func TestParse_ValidCases(t *testing.T) {
	cases := loadTestCases(t)

	for _, tc := range cases.Valid.Cases {
		t.Run(tc.File, func(t *testing.T) {
			source := loadTestFile(t, tc.File)
			dsl, result := Parse(source)

			if !result.Valid {
				t.Errorf("Expected valid DSL, got errors:\n%s", result.FormatDiagnostics())
			}

			if dsl == nil {
				t.Error("Expected non-nil DSL for valid input")
			}

			if len(result.Errors()) > 0 {
				t.Errorf("Expected no errors, got %d:\n%s", len(result.Errors()), result.FormatDiagnostics())
			}
		})
	}
}

func TestParse_InvalidCases(t *testing.T) {
	cases := loadTestCases(t)

	for _, tc := range cases.Invalid.Cases {
		t.Run(tc.File, func(t *testing.T) {
			source := loadTestFile(t, tc.File)
			dsl, result := Parse(source)

			if result.Valid {
				t.Error("Expected invalid DSL, but validation passed")
			}

			if dsl != nil {
				t.Error("Expected nil DSL for invalid input")
			}

			errors := result.Errors()
			if len(errors) == 0 {
				t.Error("Expected at least one error")
			}

			// Check expected error count
			if tc.ExpectedCount > 0 && len(errors) != tc.ExpectedCount {
				t.Errorf("Expected %d errors, got %d:\n%s",
					tc.ExpectedCount, len(errors), result.FormatDiagnostics())
			}

			if tc.ExpectedCountMin > 0 && len(errors) < tc.ExpectedCountMin {
				t.Errorf("Expected at least %d errors, got %d:\n%s",
					tc.ExpectedCountMin, len(errors), result.FormatDiagnostics())
			}

			// Check expected error codes
			if len(tc.ExpectedErrors) > 0 {
				foundCodes := make(map[string]bool)
				for _, err := range errors {
					foundCodes[err.Code] = true
				}

				for _, expectedCode := range tc.ExpectedErrors {
					if !foundCodes[expectedCode] {
						t.Errorf("Expected error code %s not found in errors:\n%s",
							expectedCode, result.FormatDiagnostics())
					}
				}
			}

			// Verify position information is present
			for _, err := range errors {
				if err.Position.Line < 1 {
					t.Errorf("Error missing valid line number: %v", err)
				}
				if err.Position.Column < 1 {
					t.Errorf("Error missing valid column number: %v", err)
				}
			}
		})
	}
}

func TestLint(t *testing.T) {
	source := `{"select": ["id", "name"], "from": "users"}`
	result := Lint(source)

	if !result.Valid {
		t.Errorf("Expected valid result, got: %s", result.FormatDiagnostics())
	}
}

func TestMustParse_Valid(t *testing.T) {
	source := `{"select": ["id", "name"], "from": "users"}`
	dsl := MustParse(source)

	if dsl == nil {
		t.Error("Expected non-nil DSL")
	}

	if len(dsl.Select) != 2 {
		t.Errorf("Expected 2 select fields, got %d", len(dsl.Select))
	}
}

func TestMustParse_Invalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for invalid DSL")
		}
	}()

	source := `{"from": "users"}` // Missing select
	MustParse(source)
}

func TestPosition_String(t *testing.T) {
	tests := []struct {
		pos      Position
		expected string
	}{
		{Position{Line: 1, Column: 5, EndLine: 1, EndColumn: 5}, "1:5"},
		{Position{Line: 1, Column: 5, EndLine: 1, EndColumn: 10}, "1:5-10"},
		{Position{Line: 1, Column: 5, EndLine: 3, EndColumn: 10}, "1:5-3:10"},
	}

	for _, tt := range tests {
		result := tt.pos.String()
		if result != tt.expected {
			t.Errorf("Position.String() = %s, expected %s", result, tt.expected)
		}
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityError, "error"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "info"},
		{SeverityHint, "hint"},
	}

	for _, tt := range tests {
		result := tt.severity.String()
		if result != tt.expected {
			t.Errorf("Severity.String() = %s, expected %s", result, tt.expected)
		}
	}
}

func TestDiagnostic_String(t *testing.T) {
	d := Diagnostic{
		Severity: SeverityError,
		Message:  "test error",
		Position: Position{Line: 5, Column: 10, EndLine: 5, EndColumn: 10},
		Path:     "wheres[0].field",
	}

	result := d.String()
	if !strings.Contains(result, "5:10") {
		t.Errorf("Diagnostic.String() missing position: %s", result)
	}
	if !strings.Contains(result, "wheres[0].field") {
		t.Errorf("Diagnostic.String() missing path: %s", result)
	}
	if !strings.Contains(result, "error") {
		t.Errorf("Diagnostic.String() missing severity: %s", result)
	}
	if !strings.Contains(result, "test error") {
		t.Errorf("Diagnostic.String() missing message: %s", result)
	}
}

func TestParse_JSONSyntaxError(t *testing.T) {
	source := `{"select": ["id", "name"`
	dsl, result := Parse(source)

	if dsl != nil {
		t.Error("Expected nil DSL for JSON syntax error")
	}

	if result.Valid {
		t.Error("Expected invalid result for JSON syntax error")
	}

	errors := result.Errors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if errors[0].Code != "E001" {
		t.Errorf("Expected error code E001, got %s", errors[0].Code)
	}
}

func TestParse_PositionTracking(t *testing.T) {
	// Multi-line source with error on specific line
	source := `{
  "select": ["id", "name"],
  "wheres": [
    { "op": "=", "value": "active" }
  ]
}`
	_, result := Parse(source)

	// Should have errors for missing from and missing field in where
	if result.Valid {
		t.Error("Expected invalid result")
	}

	// Check that position tracking works
	for _, err := range result.Errors() {
		if err.Position.Line < 1 || err.Position.Column < 1 {
			t.Errorf("Invalid position for error: %v", err)
		}
	}
}

func TestParse_ComplexNestedErrors(t *testing.T) {
	source := `{
  "select": ["id"],
  "from": "users",
  "wheres": [
    {
      "field": "status",
      "op": "=",
      "value": "active",
      "wheres": [
        { "op": ">", "value": 5 }
      ]
    }
  ]
}`
	_, result := Parse(source)

	if result.Valid {
		t.Error("Expected invalid result for nested error")
	}

	// Should find error in nested where
	found := false
	for _, err := range result.Errors() {
		if strings.Contains(err.Path, "wheres[0].wheres[0]") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected error in nested wheres path, got:\n%s", result.FormatDiagnostics())
	}
}

// Additional tests to improve coverage

func TestLintResult_Warnings(t *testing.T) {
	// Create result with mixed diagnostics
	result := &LintResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "warning1"},
			{Severity: SeverityError, Message: "error1"},
			{Severity: SeverityWarning, Message: "warning2"},
		},
	}

	warnings := result.Warnings()
	if len(warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(warnings))
	}

	for _, w := range warnings {
		if w.Severity != SeverityWarning {
			t.Errorf("Expected warning severity, got %v", w.Severity)
		}
	}
}

func TestLintResult_HasErrors(t *testing.T) {
	// Test with only warnings
	result := &LintResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "warning"},
		},
	}

	if result.HasErrors() {
		t.Error("Expected HasErrors() = false for warnings only")
	}

	// Add an error
	result.Diagnostics = append(result.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Message:  "error",
	})

	if !result.HasErrors() {
		t.Error("Expected HasErrors() = true when errors present")
	}
}

func TestLintResult_Errors(t *testing.T) {
	result := &LintResult{
		Diagnostics: []Diagnostic{
			{Severity: SeverityWarning, Message: "warning1"},
			{Severity: SeverityError, Message: "error1"},
			{Severity: SeverityInfo, Message: "info1"},
			{Severity: SeverityError, Message: "error2"},
		},
	}

	errors := result.Errors()
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
}

func TestFormatDiagnostics_Empty(t *testing.T) {
	result := &LintResult{
		Diagnostics: []Diagnostic{},
	}

	formatted := result.FormatDiagnostics()
	if formatted != "" {
		t.Errorf("Expected empty string for no diagnostics, got: %s", formatted)
	}
}

func TestSeverity_Unknown(t *testing.T) {
	var unknown Severity = 99
	if unknown.String() != "unknown" {
		t.Errorf("Expected 'unknown' for invalid severity, got: %s", unknown.String())
	}
}

func TestParse_SubQueryErrors(t *testing.T) {
	source := `{
  "select": ["id", "name"],
  "query": {
    "name": "sub",
    "from": "users"
  }
}`
	_, result := Parse(source)

	if result.Valid {
		t.Error("Expected invalid result for subquery missing select")
	}

	// Should find error in subquery
	found := false
	for _, err := range result.Errors() {
		if strings.Contains(err.Path, "query") {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected error in query path, got:\n%s", result.FormatDiagnostics())
	}
}

func TestParse_GroupsValidation(t *testing.T) {
	source := `{
  "select": ["category", ":COUNT(id) as count"],
  "from": "products",
  "groups": ["category"]
}`
	dsl, result := Parse(source)

	if !result.Valid {
		t.Errorf("Expected valid DSL, got errors:\n%s", result.FormatDiagnostics())
	}

	if dsl == nil {
		t.Error("Expected non-nil DSL")
	}
}

func TestParse_HavingNestedErrors(t *testing.T) {
	source := `{
  "select": ["category", ":COUNT(id) as count"],
  "from": "products",
  "groups": ["category"],
  "havings": [
    {
      "field": ":COUNT(id)",
      "op": ">",
      "value": 10,
      "havings": [
        { "op": "=", "value": 5 }
      ]
    }
  ]
}`
	_, result := Parse(source)

	if result.Valid {
		t.Error("Expected invalid result for nested having error")
	}
}

func TestPosition_EdgeCases(t *testing.T) {
	// Test single character position
	pos := Position{Line: 1, Column: 1, EndLine: 1, EndColumn: 1}
	str := pos.String()
	if str != "1:1" {
		t.Errorf("Expected '1:1', got '%s'", str)
	}
}

func TestParse_EmptySource(t *testing.T) {
	_, result := Parse("")

	if result.Valid {
		t.Error("Expected invalid result for empty source")
	}

	if len(result.Errors()) == 0 {
		t.Error("Expected at least one error for empty source")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	sources := []string{
		`{invalid}`,
		`{"select": }`,
		`{"select": [}`,
		`not json at all`,
	}

	for _, source := range sources {
		_, result := Parse(source)
		if result.Valid {
			t.Errorf("Expected invalid result for source: %s", source)
		}
		if len(result.Errors()) == 0 {
			t.Errorf("Expected errors for source: %s", source)
		}
	}
}
