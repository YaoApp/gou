package process_test

import (
	"os"
	"testing"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/application/disk"
	"github.com/yaoapp/gou/mcp"
)

// testAppRoot stores the application root for tests
var testAppRoot string

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Initialize application for all process tests
	testAppRoot = os.Getenv("GOU_TEST_APPLICATION")
	if testAppRoot == "" {
		testAppRoot = "../../gou-dev-app" // Default: two levels up from process package
	}

	diskApp, err := disk.Open(testAppRoot)
	if err == nil {
		application.Load(diskApp)
	}

	// Run tests
	code := m.Run()

	// Cleanup all test clients
	Clean()

	os.Exit(code)
}

// Prepare loads all test MCP clients
func Prepare(t *testing.T) {
	if application.App == nil {
		t.Skip("Application not initialized")
	}

	// Load test clients
	_, err := mcp.LoadClient("mcps/dsl.mcp.yao", "dsl")
	if err != nil {
		t.Fatalf("Failed to load dsl client: %v", err)
	}

	_, err = mcp.LoadClient("mcps/echo.mcp.yao", "echo")
	if err != nil {
		t.Fatalf("Failed to load echo client: %v", err)
	}

	_, err = mcp.LoadClient("mcps/customer.mcp.yao", "customer")
	if err != nil {
		t.Fatalf("Failed to load customer client: %v", err)
	}
}

// Clean unloads all test MCP clients
func Clean() {
	mcp.UnloadClient("dsl")
	mcp.UnloadClient("echo")
	mcp.UnloadClient("customer")
}
