package process

import (
	"context"
)

// Process the process sturct
type Process struct {
	Name       string
	Group      string
	Method     string
	Handler    string
	ID         string
	Args       []interface{}
	Global     map[string]interface{} // Global vars
	Sid        string                 // Session ID
	Context    context.Context        // Context
	V8Context  interface{}            `json:"-"` // V8 Context (for thread affinity in JavaScript calls)
	Runtime    Runtime                `json:"-"` // Runtime
	Callback   CallbackFunc           `json:"-"` // Callback
	Authorized *AuthorizedInfo        `json:"-"` // Authorized information (set by OAuth guard)
	_val       *interface{}           // Value // The result of the process
}

// AuthorizedInfo represents authorized information for the process
// This structure matches the AuthorizedInfo from the OAuth system
type AuthorizedInfo struct {
	Subject   string `json:"sub,omitempty"`        // Subject identifier
	ClientID  string `json:"client_id"`            // OAuth client ID
	Scope     string `json:"scope,omitempty"`      // Access scope
	SessionID string `json:"session_id,omitempty"` // Session ID
	UserID    string `json:"user_id,omitempty"`    // User ID

	// Extended fields for multi-tenancy and team support
	TeamID     string `json:"team_id,omitempty"`     // Team identifier
	TenantID   string `json:"tenant_id,omitempty"`   // Tenant identifier
	RememberMe bool   `json:"remember_me,omitempty"` // Remember Me flag preserved from login

	// Data access constraints (set by ACL enforcement)
	Constraints DataConstraints `json:"constraints,omitempty"`
}

// DataConstraints represents data access constraints for the process
type DataConstraints struct {
	// Built-in constraints
	OwnerOnly   bool // Only access owner's data (current owner)
	CreatorOnly bool // Only access creator's data (who created the resource)
	EditorOnly  bool // Only access editor's data (who last updated the resource)
	TeamOnly    bool // Only access team's data (filter by TeamID)

	// Extra constraints (user-defined, flexible extension)
	Extra map[string]interface{} // Custom constraints like department_only, region_only, etc.
}

// CallbackFunc the callback function
type CallbackFunc func(process *Process, data map[string]interface{}) error

// Runtime interface
type Runtime interface {
	Dispose()
}

// Handler the process handler
type Handler func(process *Process) interface{}
