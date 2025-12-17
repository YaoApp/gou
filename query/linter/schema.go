package linter

import (
	"encoding/json"

	goujson "github.com/yaoapp/gou/json"
)

// QueryDSLSchemaJSON is the JSON Schema for QueryDSL as a string constant
// This is useful for MCP/LLM tool calls and validation
//
// Expression Syntax (string):
//   - field: "name", "table.name", "$model.name"
//   - alias: "name as n", "field as 别名"
//   - function: ":MAX(field)", ":COUNT(id)", ":SUM(amount)"
//   - constant: "123", "0.618", "'string'"
//   - encrypted: "field*"
//   - object: "object$.foo", "object$.arr[0]"
//   - array: "array[0]", "array[*]", "array@"
//   - binding: "?:name", "?:extra.score"
//   - type: "field(string 50)", "price(decimal 11,2)"
//
// Table Syntax (string):
//   - table: "users", "users as u"
//   - model: "$user", "$user as u"
//
// Condition Syntax (object):
//   - standard: { "field": "score", "op": "=", "value": 20 }
//   - shorthand: { "field": "score", "=": 20 }
//   - field shorthand: { ":score": "comment", "=": 20 }
//   - or condition: { "or": true, "field": "name", "=": "test" }
//   - or shorthand: { "or :name": "comment", "=": "test" }
//   - value expression: { "field": "a", "=": "{b}" }
//
// Orders Syntax:
//   - string array: ["id desc", "name asc"]
//   - object array: [{"field": "id", "sort": "desc"}]
//
// Groups Syntax:
//   - string: "kind, city rollup 所有城市"
//   - string array: ["kind", "city rollup 所有城市"]
//   - object array: [{"field": "kind"}, {"field": "city", "rollup": "所有城市"}]
const QueryDSLSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "QueryDSL",
  "description": "Gou Query Domain Specific Language for database queries",
  "type": "object",
  "definitions": {
    "expression": {
      "type": "string",
      "description": "Field expression. Syntax: field, table.field, $model.field, :FUNC(args), field as alias, 'const', 123, field*, object$.key, array[0], ?:binding, field(type)"
    },
    "table": {
      "type": "string",
      "description": "Table or model reference. Syntax: table, table as alias, $model, $model as alias"
    },
    "condition": {
      "type": "object",
      "description": "Query condition. Standard: {field, op, value}. Shorthand: {field, '=': value} or {':field': comment, '=': value}",
      "properties": {
        "field": { "type": "string", "description": "Field expression" },
        "op": { "type": "string", "description": "Operator: =, >, >=, <, <=, <>, like, match, in, is" },
        "value": { "description": "Compare value. Use {field} for field reference" },
        "or": { "type": "boolean", "description": "Use OR instead of AND", "default": false },
        "query": { "$ref": "#", "description": "Subquery for IN clause" },
        "comment": { "type": "string", "description": "Condition comment" },
        "=": { "description": "Shorthand for op='=' with value" },
        ">": { "description": "Shorthand for op='>' with value" },
        ">=": { "description": "Shorthand for op='>=' with value" },
        "<": { "description": "Shorthand for op='<' with value" },
        "<=": { "description": "Shorthand for op='<=' with value" },
        "<>": { "description": "Shorthand for op='<>' with value" },
        "like": { "description": "Shorthand for op='like' with value" },
        "match": { "description": "Shorthand for op='match' with value" },
        "in": { "type": "array", "description": "Shorthand for op='in' with value array" },
        "is": { "type": "string", "enum": ["null", "not null"], "description": "Shorthand for op='is' with null check" }
      },
      "additionalProperties": true
    },
    "where": {
      "allOf": [
        { "$ref": "#/definitions/condition" },
        {
          "type": "object",
          "properties": {
            "wheres": {
              "type": "array",
              "description": "Nested conditions for grouping: cond1 AND (cond2 OR cond3)",
              "items": { "$ref": "#/definitions/where" }
            }
          }
        }
      ]
    },
    "order": {
      "oneOf": [
        { "type": "string", "description": "Order expression: 'field desc', 'field asc', ':MAX(id) desc'" },
        {
          "type": "object",
          "properties": {
            "field": { "$ref": "#/definitions/expression" },
            "sort": { "type": "string", "enum": ["asc", "desc"], "default": "asc" },
            "comment": { "type": "string" }
          },
          "required": ["field"]
        }
      ]
    },
    "group": {
      "oneOf": [
        { "type": "string", "description": "Group expression: 'field', 'field rollup 合计'" },
        {
          "type": "object",
          "properties": {
            "field": { "$ref": "#/definitions/expression" },
            "rollup": { "type": "string", "description": "Rollup field name for subtotals" },
            "comment": { "type": "string" }
          },
          "required": ["field"]
        }
      ]
    },
    "having": {
      "allOf": [
        { "$ref": "#/definitions/condition" },
        {
          "type": "object",
          "properties": {
            "havings": {
              "type": "array",
              "description": "Nested having conditions",
              "items": { "$ref": "#/definitions/having" }
            }
          }
        }
      ]
    },
    "join": {
      "type": "object",
      "description": "Table join specification",
      "properties": {
        "from": { "$ref": "#/definitions/table", "description": "Table to join" },
        "key": { "$ref": "#/definitions/expression", "description": "Join key field" },
        "foreign": { "$ref": "#/definitions/expression", "description": "Foreign key field" },
        "left": { "type": "boolean", "description": "LEFT JOIN", "default": false },
        "right": { "type": "boolean", "description": "RIGHT JOIN", "default": false },
        "select": { "type": "array", "items": { "$ref": "#/definitions/expression" }, "description": "Fields to select from joined table" },
        "comment": { "type": "string" }
      },
      "required": ["from", "key", "foreign"]
    },
    "sql": {
      "type": "object",
      "description": "Raw SQL statement",
      "properties": {
        "stmt": { "type": "string", "description": "SQL statement with ? placeholders" },
        "args": { "type": "array", "description": "Bind parameter values" },
        "comment": { "type": "string" }
      },
      "required": ["stmt"]
    }
  },
  "properties": {
    "select": {
      "type": "array",
      "description": "Fields to select",
      "items": { "$ref": "#/definitions/expression" }
    },
    "from": {
      "$ref": "#/definitions/table",
      "description": "Table or model to query"
    },
    "wheres": {
      "type": "array",
      "description": "WHERE conditions",
      "items": { "$ref": "#/definitions/where" }
    },
    "orders": {
      "oneOf": [
        { "type": "array", "items": { "$ref": "#/definitions/order" } },
        { "type": "string", "description": "Comma-separated: 'id desc, name asc'" }
      ],
      "description": "ORDER BY clause"
    },
    "groups": {
      "oneOf": [
        { "type": "array", "items": { "$ref": "#/definitions/group" } },
        { "type": "string", "description": "Comma-separated: 'kind, city rollup 所有城市'" }
      ],
      "description": "GROUP BY clause"
    },
    "havings": {
      "type": "array",
      "description": "HAVING conditions (requires groups)",
      "items": { "$ref": "#/definitions/having" }
    },
    "joins": {
      "type": "array",
      "description": "JOIN clauses",
      "items": { "$ref": "#/definitions/join" }
    },
    "unions": {
      "type": "array",
      "description": "UNION queries",
      "items": { "$ref": "#" }
    },
    "query": {
      "$ref": "#",
      "description": "Subquery (alternative to from)"
    },
    "name": {
      "type": "string",
      "description": "Subquery alias"
    },
    "sql": {
      "$ref": "#/definitions/sql",
      "description": "Raw SQL (alternative to select/from)"
    },
    "first": {
      "oneOf": [{ "type": "boolean" }, { "type": "integer" }],
      "description": "Return first record(s)"
    },
    "limit": {
      "oneOf": [{ "type": "integer" }, { "type": "string" }],
      "description": "Maximum records to return"
    },
    "offset": {
      "oneOf": [{ "type": "integer" }, { "type": "string" }],
      "description": "Records to skip"
    },
    "page": {
      "oneOf": [{ "type": "integer" }, { "type": "string" }],
      "description": "Page number (1-based)"
    },
    "pagesize": {
      "oneOf": [{ "type": "integer" }, { "type": "string" }],
      "description": "Records per page"
    },
    "data-only": {
      "oneOf": [{ "type": "boolean" }, { "type": "string" }],
      "description": "Return data only without pagination info"
    },
    "comment": {
      "type": "string",
      "description": "Query comment"
    },
    "debug": {
      "type": "boolean",
      "description": "Enable debug logging"
    }
  }
}`

// QueryDSLSchema returns the JSON Schema for QueryDSL as a map
func QueryDSLSchema() map[string]interface{} {
	var schema map[string]interface{}
	json.Unmarshal([]byte(QueryDSLSchemaJSON), &schema)
	return schema
}

// Validator returns a JSON Schema validator for QueryDSL
func Validator() (*goujson.Validator, error) {
	return goujson.NewValidator(QueryDSLSchemaJSON)
}

// ValidateSchema validates data against the QueryDSL JSON Schema
// Returns nil if valid, error with validation details otherwise
func ValidateSchema(data interface{}) error {
	return goujson.Validate(data, QueryDSLSchemaJSON)
}
