package linter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/exception"
)

// Severity represents the severity level of a diagnostic
type Severity int

const (
	// SeverityError indicates an error that prevents query execution
	SeverityError Severity = iota
	// SeverityWarning indicates a potential issue that might cause problems
	SeverityWarning
	// SeverityInfo indicates informational messages
	SeverityInfo
	// SeverityHint indicates suggestions for improvement
	SeverityHint
)

// String returns the string representation of severity
func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "info"
	case SeverityHint:
		return "hint"
	default:
		return "unknown"
	}
}

// Position represents a location in the source DSL
type Position struct {
	Line      int `json:"line"`       // 1-based line number
	Column    int `json:"column"`     // 1-based column number
	Offset    int `json:"offset"`     // 0-based byte offset
	EndLine   int `json:"end_line"`   // 1-based end line number
	EndColumn int `json:"end_column"` // 1-based end column number
	EndOffset int `json:"end_offset"` // 0-based end byte offset
}

// String returns a human-readable position string
func (p Position) String() string {
	if p.Line == p.EndLine && p.Column == p.EndColumn {
		return fmt.Sprintf("%d:%d", p.Line, p.Column)
	}
	if p.Line == p.EndLine {
		return fmt.Sprintf("%d:%d-%d", p.Line, p.Column, p.EndColumn)
	}
	return fmt.Sprintf("%d:%d-%d:%d", p.Line, p.Column, p.EndLine, p.EndColumn)
}

// Diagnostic represents a single validation error or warning
type Diagnostic struct {
	Severity Severity `json:"severity"` // Error severity level
	Message  string   `json:"message"`  // Human-readable error message
	Position Position `json:"position"` // Location in source
	Path     string   `json:"path"`     // JSON path to the error (e.g., "wheres[0].field")
	Code     string   `json:"code"`     // Error code for programmatic handling
	Source   string   `json:"source"`   // The problematic source text
}

// String returns a human-readable diagnostic string
func (d Diagnostic) String() string {
	return fmt.Sprintf("%s:%s: %s: %s", d.Position.String(), d.Path, d.Severity.String(), d.Message)
}

// LintResult contains the complete result of parsing and validating a DSL
type LintResult struct {
	DSL         *gou.QueryDSL `json:"dsl,omitempty"` // Parsed DSL (nil if parsing failed)
	Diagnostics []Diagnostic  `json:"diagnostics"`   // All diagnostics found
	Source      string        `json:"-"`             // Original source
	Valid       bool          `json:"valid"`         // True if no errors (warnings allowed)
	lineOffsets []int         // Cache of line start offsets for position calculation
}

// HasErrors returns true if there are any error-level diagnostics
func (r *LintResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns only error-level diagnostics
func (r *LintResult) Errors() []Diagnostic {
	var errors []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			errors = append(errors, d)
		}
	}
	return errors
}

// Warnings returns only warning-level diagnostics
func (r *LintResult) Warnings() []Diagnostic {
	var warnings []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityWarning {
			warnings = append(warnings, d)
		}
	}
	return warnings
}

// FormatDiagnostics returns a formatted string of all diagnostics
func (r *LintResult) FormatDiagnostics() string {
	if len(r.Diagnostics) == 0 {
		return ""
	}

	var sb strings.Builder
	for i, d := range r.Diagnostics {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(d.String())
	}
	return sb.String()
}

// Parse parses a JSON DSL string and returns the parsed DSL and LintResult.
// If validation passes (no errors), the returned DSL can be used directly.
// Returns: (*gou.QueryDSL, *LintResult) - DSL is nil if parsing/validation failed
func Parse(source string) (dsl *gou.QueryDSL, result *LintResult) {
	result = &LintResult{
		Source:      source,
		Diagnostics: []Diagnostic{},
		Valid:       true,
	}

	// Build line offset cache
	result.buildLineOffsets()

	// Recover from panics during JSON unmarshalling (e.g., invalid expressions)
	defer func() {
		if r := recover(); r != nil {
			var errMsg string
			switch v := r.(type) {
			case string:
				errMsg = v
			case error:
				errMsg = v.Error()
			case exception.Exception:
				errMsg = v.Message
			case *exception.Exception:
				errMsg = v.Message
			default:
				errMsg = fmt.Sprintf("%v", v)
			}
			result.Diagnostics = append(result.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("Parse error: %s", errMsg),
				Position: Position{Line: 1, Column: 1, EndLine: 1, EndColumn: 1},
				Path:     "",
				Code:     "E002",
			})
			result.Valid = false
			dsl = nil
		}
	}()

	// Try to parse JSON
	var parsedDSL gou.QueryDSL
	if err := json.Unmarshal([]byte(source), &parsedDSL); err != nil {
		// Parse JSON error and extract position
		result.addJSONError(err)
		result.Valid = false
		return nil, result
	}

	result.DSL = &parsedDSL

	// Validate the parsed DSL with position tracking
	result.validateDSL(source)

	// Update valid status based on errors
	result.Valid = !result.HasErrors()

	// Return DSL only if validation passed
	if result.Valid {
		return &parsedDSL, result
	}
	return nil, result
}

// Lint validates a DSL string and returns only the LintResult.
// Use Parse() if you need both the DSL and validation result.
func Lint(source string) *LintResult {
	_, result := Parse(source)
	return result
}

// MustParse parses a DSL string and panics if validation fails.
// Returns the parsed QueryDSL directly for convenience.
func MustParse(source string) *gou.QueryDSL {
	dsl, result := Parse(source)
	if !result.Valid {
		panic(fmt.Sprintf("QueryDSL validation failed:\n%s", result.FormatDiagnostics()))
	}
	return dsl
}

// buildLineOffsets builds a cache of line start offsets
func (r *LintResult) buildLineOffsets() {
	r.lineOffsets = []int{0} // First line starts at offset 0
	for i, ch := range r.Source {
		if ch == '\n' {
			r.lineOffsets = append(r.lineOffsets, i+1)
		}
	}
}

// offsetToPosition converts a byte offset to line and column
func (r *LintResult) offsetToPosition(offset int) (line, column int) {
	if len(r.lineOffsets) == 0 {
		return 1, offset + 1
	}

	// Binary search for the line
	line = sort.Search(len(r.lineOffsets), func(i int) bool {
		return r.lineOffsets[i] > offset
	})

	if line == 0 {
		line = 1
	}

	// Calculate column
	lineStart := r.lineOffsets[line-1]
	column = offset - lineStart + 1

	return line, column
}

// addJSONError adds a diagnostic for a JSON parsing error
func (r *LintResult) addJSONError(err error) {
	errStr := err.Error()

	// Try to extract position from JSON error message
	// Format: "invalid character 'x' at offset 123"
	// or: "unexpected end of JSON input"
	var pos Position
	pos.Line = 1
	pos.Column = 1

	// Try to find offset in error message
	offsetRe := regexp.MustCompile(`offset\s+(\d+)`)
	if matches := offsetRe.FindStringSubmatch(errStr); len(matches) > 1 {
		if offset, e := strconv.Atoi(matches[1]); e == nil {
			line, col := r.offsetToPosition(offset)
			pos.Line = line
			pos.Column = col
			pos.Offset = offset
			pos.EndLine = line
			pos.EndColumn = col + 1
			pos.EndOffset = offset + 1
		}
	}

	r.Diagnostics = append(r.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Message:  fmt.Sprintf("JSON parse error: %s", errStr),
		Position: pos,
		Path:     "",
		Code:     "E001",
		Source:   r.extractSourceContext(pos.Offset, 20),
	})
}

// extractSourceContext extracts a context string around an offset
func (r *LintResult) extractSourceContext(offset, length int) string {
	if offset < 0 || offset >= len(r.Source) {
		return ""
	}

	start := offset
	end := offset + length
	if end > len(r.Source) {
		end = len(r.Source)
	}

	return r.Source[start:end]
}

// validateDSL validates the parsed DSL and adds diagnostics with position info
func (r *LintResult) validateDSL(source string) {
	if r.DSL == nil {
		return
	}

	// Validate select
	r.validateSelect(source)

	// Validate from
	r.validateFrom(source)

	// Validate wheres
	r.validateWheres(source)

	// Validate orders
	r.validateOrders(source)

	// Validate groups
	r.validateGroups(source)

	// Validate havings
	r.validateHavings(source)

	// Validate unions
	r.validateUnions(source)

	// Validate joins
	r.validateJoins(source)

	// Validate SQL
	r.validateSQL(source)

	// Validate subquery
	r.validateSubQuery(source)
}

// findFieldPosition finds the position of a field in the source
func (r *LintResult) findFieldPosition(fieldPath string, source string) Position {
	// Convert path like "wheres[0].field" to a search pattern
	parts := strings.Split(fieldPath, ".")

	// Start with the full source
	searchStart := 0

	for i, part := range parts {
		// Handle array index
		if idx := strings.Index(part, "["); idx != -1 {
			arrayName := part[:idx]
			// Find the array key
			pattern := fmt.Sprintf(`"%s"\s*:\s*\[`, regexp.QuoteMeta(arrayName))
			re := regexp.MustCompile(pattern)
			loc := re.FindStringIndex(source[searchStart:])
			if loc != nil {
				searchStart += loc[1]
			}

			// Extract index
			idxEnd := strings.Index(part[idx:], "]")
			if idxEnd != -1 {
				indexStr := part[idx+1 : idx+idxEnd]
				if index, err := strconv.Atoi(indexStr); err == nil {
					// Skip to the nth element
					braceCount := 0
					elementCount := 0
					for j := searchStart; j < len(source); j++ {
						ch := source[j]
						if ch == '{' || ch == '[' {
							if braceCount == 0 && ch == '{' {
								if elementCount == index {
									searchStart = j
									break
								}
							}
							braceCount++
						} else if ch == '}' || ch == ']' {
							braceCount--
						} else if ch == ',' && braceCount == 0 {
							elementCount++
						}
					}
				}
			}
		} else if i < len(parts)-1 {
			// Find the key
			pattern := fmt.Sprintf(`"%s"\s*:`, regexp.QuoteMeta(part))
			re := regexp.MustCompile(pattern)
			loc := re.FindStringIndex(source[searchStart:])
			if loc != nil {
				searchStart += loc[1]
			}
		} else {
			// Last part - find the actual field
			pattern := fmt.Sprintf(`"%s"\s*:`, regexp.QuoteMeta(part))
			re := regexp.MustCompile(pattern)
			loc := re.FindStringIndex(source[searchStart:])
			if loc != nil {
				offset := searchStart + loc[0]
				line, col := r.offsetToPosition(offset)
				endLine, endCol := r.offsetToPosition(searchStart + loc[1])
				return Position{
					Line:      line,
					Column:    col,
					Offset:    offset,
					EndLine:   endLine,
					EndColumn: endCol,
					EndOffset: searchStart + loc[1],
				}
			}
		}
	}

	// Fallback: search for the last part directly
	lastPart := parts[len(parts)-1]
	if idx := strings.Index(lastPart, "["); idx != -1 {
		lastPart = lastPart[:idx]
	}

	pattern := fmt.Sprintf(`"%s"`, regexp.QuoteMeta(lastPart))
	re := regexp.MustCompile(pattern)
	loc := re.FindStringIndex(source)
	if loc != nil {
		line, col := r.offsetToPosition(loc[0])
		endLine, endCol := r.offsetToPosition(loc[1])
		return Position{
			Line:      line,
			Column:    col,
			Offset:    loc[0],
			EndLine:   endLine,
			EndColumn: endCol,
			EndOffset: loc[1],
		}
	}

	return Position{Line: 1, Column: 1, EndLine: 1, EndColumn: 1}
}

// validateSelect validates the select clause
func (r *LintResult) validateSelect(source string) {
	if r.DSL.Select == nil && r.DSL.SQL == nil {
		pos := r.findFieldPosition("select", source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "Missing required field: select or sql must be specified",
			Position: pos,
			Path:     "select",
			Code:     "E100",
		})
		return
	}

	for i, exp := range r.DSL.Select {
		path := fmt.Sprintf("select[%d]", i)
		if err := exp.Validate(); err != nil {
			pos := r.findFieldPosition(path, source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("Invalid expression: %s", err.Error()),
				Position: pos,
				Path:     path,
				Code:     "E101",
				Source:   exp.Origin,
			})
		}
	}
}

// validateFrom validates the from clause
func (r *LintResult) validateFrom(source string) {
	// Skip from validation if SQL is specified
	if r.DSL.SQL != nil {
		return
	}

	if r.DSL.SubQuery == nil && r.DSL.From == nil {
		pos := r.findFieldPosition("from", source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "Missing required field: from or query must be specified",
			Position: pos,
			Path:     "from",
			Code:     "E110",
		})
		return
	}

	if r.DSL.From != nil {
		if err := r.DSL.From.Validate(); err != nil {
			pos := r.findFieldPosition("from", source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  fmt.Sprintf("Invalid from: %s", err.Error()),
				Position: pos,
				Path:     "from",
				Code:     "E111",
				Source:   r.DSL.From.ToString(),
			})
		}
	}
}

// validateWheres validates the wheres clause
func (r *LintResult) validateWheres(source string) {
	if r.DSL.Wheres == nil {
		return
	}

	for i, where := range r.DSL.Wheres {
		r.validateWhere(where, fmt.Sprintf("wheres[%d]", i), source)
	}
}

// validateWhere validates a single where condition
func (r *LintResult) validateWhere(where gou.Where, path string, source string) {
	errs := where.Condition.Validate()
	for _, err := range errs {
		pos := r.findFieldPosition(path, source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  err.Error(),
			Position: pos,
			Path:     path,
			Code:     "E120",
		})
	}

	// Validate nested wheres
	for i, nestedWhere := range where.Wheres {
		r.validateWhere(nestedWhere, fmt.Sprintf("%s.wheres[%d]", path, i), source)
	}
}

// validateOrders validates the orders clause
func (r *LintResult) validateOrders(source string) {
	if r.DSL.Orders == nil {
		return
	}

	for i, order := range r.DSL.Orders {
		path := fmt.Sprintf("orders[%d]", i)
		if err := order.Validate(); err != nil {
			pos := r.findFieldPosition(path, source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  err.Error(),
				Position: pos,
				Path:     path,
				Code:     "E130",
			})
		}
	}
}

// validateGroups validates the groups clause
func (r *LintResult) validateGroups(source string) {
	if r.DSL.Groups == nil {
		return
	}

	errs := r.DSL.Groups.Validate()
	for i, err := range errs {
		path := fmt.Sprintf("groups[%d]", i)
		pos := r.findFieldPosition(path, source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  err.Error(),
			Position: pos,
			Path:     path,
			Code:     "E140",
		})
	}
}

// validateHavings validates the havings clause
func (r *LintResult) validateHavings(source string) {
	if len(r.DSL.Havings) > 0 && r.DSL.Groups == nil {
		pos := r.findFieldPosition("havings", source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "havings requires groups to be specified",
			Position: pos,
			Path:     "havings",
			Code:     "E150",
		})
	}

	for i, having := range r.DSL.Havings {
		r.validateHaving(having, fmt.Sprintf("havings[%d]", i), source)
	}
}

// validateHaving validates a single having condition
func (r *LintResult) validateHaving(having gou.Having, path string, source string) {
	errs := having.Condition.Validate()
	for _, err := range errs {
		pos := r.findFieldPosition(path, source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  err.Error(),
			Position: pos,
			Path:     path,
			Code:     "E151",
		})
	}

	// Validate nested havings
	for i, nestedHaving := range having.Havings {
		r.validateHaving(nestedHaving, fmt.Sprintf("%s.havings[%d]", path, i), source)
	}
}

// validateUnions validates the unions clause
func (r *LintResult) validateUnions(source string) {
	if r.DSL.Unions == nil {
		return
	}

	for i, union := range r.DSL.Unions {
		path := fmt.Sprintf("unions[%d]", i)
		errs := union.Validate()
		for _, err := range errs {
			pos := r.findFieldPosition(path, source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  err.Error(),
				Position: pos,
				Path:     path,
				Code:     "E160",
			})
		}
	}
}

// validateJoins validates the joins clause
func (r *LintResult) validateJoins(source string) {
	if r.DSL.Joins == nil {
		return
	}

	for i, join := range r.DSL.Joins {
		path := fmt.Sprintf("joins[%d]", i)

		if join.Key == nil {
			pos := r.findFieldPosition(path+".key", source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  "Missing required field: key",
				Position: pos,
				Path:     path + ".key",
				Code:     "E170",
			})
		}

		if join.Foreign == nil {
			pos := r.findFieldPosition(path+".foreign", source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  "Missing required field: foreign",
				Position: pos,
				Path:     path + ".foreign",
				Code:     "E171",
			})
		}

		if join.From == nil {
			pos := r.findFieldPosition(path+".from", source)
			r.Diagnostics = append(r.Diagnostics, Diagnostic{
				Severity: SeverityError,
				Message:  "Missing required field: from",
				Position: pos,
				Path:     path + ".from",
				Code:     "E172",
			})
		}
	}
}

// validateSQL validates the sql clause
func (r *LintResult) validateSQL(source string) {
	if r.DSL.SQL == nil {
		return
	}

	if r.DSL.SQL.STMT == "" {
		pos := r.findFieldPosition("sql.stmt", source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  "Missing required field: sql.stmt",
			Position: pos,
			Path:     "sql.stmt",
			Code:     "E180",
		})
	}
}

// validateSubQuery validates the subquery
func (r *LintResult) validateSubQuery(source string) {
	if r.DSL.SubQuery == nil {
		return
	}

	errs := r.DSL.SubQuery.Validate()
	for _, err := range errs {
		pos := r.findFieldPosition("query", source)
		r.Diagnostics = append(r.Diagnostics, Diagnostic{
			Severity: SeverityError,
			Message:  err.Error(),
			Position: pos,
			Path:     "query",
			Code:     "E190",
		})
	}
}
