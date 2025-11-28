package process_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/mcp"
)

func TestMappingData(t *testing.T) {
	Prepare(t)
	defer Clean()

	t.Run("should have correct tool process mappings", func(t *testing.T) {
		// Get DSL client mapping
		mapping, err := mcp.GetClientMapping("dsl")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify tool process mappings
		assert.Equal(t, "scripts.dsl.ValidateModel", mapping.Tools["validate_model"].Process)
		assert.Equal(t, "scripts.dsl.FormatFlow", mapping.Tools["format_flow"].Process)
		assert.Equal(t, "scripts.dsl.AnalyzeAPI", mapping.Tools["analyze_api"].Process)

		// Verify schemas are loaded
		for name, tool := range mapping.Tools {
			assert.NotEmpty(t, tool.InputSchema, "Tool %s should have input schema", name)
		}
	})

	t.Run("should have correct resource process mappings", func(t *testing.T) {
		// Get Customer client mapping
		mapping, err := mcp.GetClientMapping("customer")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify resource process mappings
		assert.Equal(t, "models.customer.Find", mapping.Resources["detail"].Process)
		assert.Equal(t, "models.customer.Paginate", mapping.Resources["list"].Process)

		// Verify resource URIs
		assert.Equal(t, "customers://{id}", mapping.Resources["detail"].URI)
		assert.Equal(t, "customers://list", mapping.Resources["list"].URI)

		// Verify list resource has parameters
		listRes := mapping.Resources["list"]
		assert.NotNil(t, listRes.Parameters)
		assert.Equal(t, 4, len(listRes.Parameters), "List resource should have 4 parameters")

		// Verify parameter names
		paramNames := make(map[string]bool)
		for _, param := range listRes.Parameters {
			paramNames[param.Name] = true
		}
		assert.True(t, paramNames["page"], "Should have page parameter")
		assert.True(t, paramNames["pagesize"], "Should have pagesize parameter")
		assert.True(t, paramNames["where"], "Should have where parameter")
		assert.True(t, paramNames["order"], "Should have order parameter")
	})

	t.Run("should have correct echo tool mappings", func(t *testing.T) {
		// Get Echo client mapping
		mapping, err := mcp.GetClientMapping("echo")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify echo tool process mappings
		assert.Equal(t, "scripts.echo.Ping", mapping.Tools["ping"].Process)
		assert.Equal(t, "scripts.echo.Status", mapping.Tools["status"].Process)

		// Verify ping has both input and output schema
		pingTool := mapping.Tools["ping"]
		assert.NotEmpty(t, pingTool.InputSchema, "Ping should have input schema")
		assert.NotEmpty(t, pingTool.OutputSchema, "Ping should have output schema")
	})

	t.Run("should have correct customer tool mappings", func(t *testing.T) {
		// Get Customer client mapping
		mapping, err := mcp.GetClientMapping("customer")
		assert.NoError(t, err)
		assert.NotNil(t, mapping)

		// Verify customer tool process mappings
		assert.Equal(t, "models.customer.Create", mapping.Tools["create_customer"].Process)
		assert.Equal(t, "models.customer.Update", mapping.Tools["update_customer"].Process)

		// Verify create_customer has output schema
		createTool := mapping.Tools["create_customer"]
		assert.NotEmpty(t, createTool.InputSchema)
		assert.NotEmpty(t, createTool.OutputSchema, "create_customer should have output schema")
	})
}
