# QueryDSL Linter

A validation and parsing library for Yao QueryDSL with precise error location reporting, similar to a code linter.

## Overview

This package provides:
- **Parsing** - Parse JSON DSL strings into `gou.QueryDSL` structs
- **Validation** - Validate DSL structure with detailed error diagnostics
- **Position Tracking** - Report exact line/column positions for errors
- **Error Codes** - Programmatic error codes for automated handling

## Installation

```go
import "github.com/yaoapp/gou/query/linter"
```

## Usage

### Parse and Validate

```go
// Parse returns both the DSL and validation result
dsl, result := linter.Parse(source)

if result.Valid {
    // DSL is ready to use
    fmt.Println("Select fields:", dsl.Select)
} else {
    // Handle errors
    fmt.Println(result.FormatDiagnostics())
}
```

### Validate Only

```go
// Lint returns only the validation result
result := linter.Lint(source)

if result.HasErrors() {
    for _, err := range result.Errors() {
        fmt.Printf("Line %d, Col %d: %s\n", 
            err.Position.Line, 
            err.Position.Column, 
            err.Message)
    }
}
```

### Parse with Panic on Error

```go
// MustParse panics if validation fails - useful for tests
dsl := linter.MustParse(source)
```

## Types

### LintResult

```go
type LintResult struct {
    DSL         *gou.QueryDSL // Parsed DSL (nil if failed)
    Diagnostics []Diagnostic  // All diagnostics found
    Valid       bool          // True if no errors
}

// Methods
result.HasErrors() bool           // Check if any errors exist
result.Errors() []Diagnostic      // Get only error-level diagnostics
result.Warnings() []Diagnostic    // Get only warning-level diagnostics
result.FormatDiagnostics() string // Format all diagnostics as string
```

### Diagnostic

```go
type Diagnostic struct {
    Severity Severity // SeverityError, SeverityWarning, SeverityInfo, SeverityHint
    Message  string   // Human-readable error message
    Position Position // Location in source
    Path     string   // JSON path (e.g., "wheres[0].field")
    Code     string   // Error code (e.g., "E120")
    Source   string   // Problematic source text
}
```

### Position

```go
type Position struct {
    Line      int // 1-based line number
    Column    int // 1-based column number
    Offset    int // 0-based byte offset
    EndLine   int // 1-based end line
    EndColumn int // 1-based end column
    EndOffset int // 0-based end byte offset
}
```

## Error Codes

| Code | Description |
|------|-------------|
| E001 | JSON parse error |
| E002 | Expression parse error (panic recovered) |
| E100 | Missing select field |
| E101 | Invalid select expression |
| E110 | Missing from field |
| E111 | Invalid from/table name |
| E120 | Invalid where condition |
| E130 | Invalid order specification |
| E140 | Invalid group specification |
| E150 | Havings without groups |
| E151 | Invalid having condition |
| E160 | Invalid union query |
| E170 | Join missing key |
| E171 | Join missing foreign |
| E172 | Join missing from |
| E180 | SQL missing stmt |
| E190 | Invalid subquery |

## Example Output

```
3:5-12:wheres[0]: error: Missing required field: field
5:3-8:orders[1]: error: Invalid sort value, must be 'asc' or 'desc'
```

## Integration with AI Code Generation

When generating QueryDSL:

1. **Always validate** generated DSL before use:
```go
dsl, result := linter.Parse(generatedJSON)
if !result.Valid {
    // Return errors to AI for correction
    return result.FormatDiagnostics()
}
```

2. **Use error codes** for programmatic handling:
```go
for _, err := range result.Errors() {
    switch err.Code {
    case "E120":
        // Fix where condition
    case "E130":
        // Fix order specification
    }
}
```

3. **Leverage position info** for precise fixes:
```go
for _, err := range result.Errors() {
    fmt.Printf("Fix at line %d: %s\n", err.Position.Line, err.Message)
}
```

## Test Assets

Test cases are located in `../assets/linter/`:
- `valid/` - Valid DSL examples
- `invalid/` - Invalid DSL examples with expected errors
- `testcases.json` - Test case metadata

## Coverage

Current test coverage: **92.4%**

