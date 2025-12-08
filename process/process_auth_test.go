package process

import (
	"testing"
)

func TestProcess_WithAuthorized(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected *AuthorizedInfo
	}{
		{
			name: "WithAuthorizedInfo struct",
			input: &AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				Scope:      "read write",
				SessionID:  "session789",
				UserID:     "user123",
				TeamID:     "team456",
				TenantID:   "tenant789",
				RememberMe: true,
				Constraints: DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
			expected: &AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				Scope:      "read write",
				SessionID:  "session789",
				UserID:     "user123",
				TeamID:     "team456",
				TenantID:   "tenant789",
				RememberMe: true,
				Constraints: DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
		},
		{
			name: "WithAuthorized from map - full fields",
			input: map[string]interface{}{
				"sub":         "user123",
				"client_id":   "client456",
				"scope":       "read write",
				"session_id":  "session789",
				"user_id":     "user123",
				"team_id":     "team456",
				"tenant_id":   "tenant789",
				"remember_me": true,
				"constraints": map[string]interface{}{
					"owner_only":   true,
					"creator_only": false,
					"editor_only":  false,
					"team_only":    true,
					"extra": map[string]interface{}{
						"department": "engineering",
					},
				},
			},
			expected: &AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				Scope:      "read write",
				SessionID:  "session789",
				UserID:     "user123",
				TeamID:     "team456",
				TenantID:   "tenant789",
				RememberMe: true,
				Constraints: DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
		},
		{
			name: "WithAuthorized from map - partial fields",
			input: map[string]interface{}{
				"user_id": "user123",
				"team_id": "team456",
				"scope":   "read",
			},
			expected: &AuthorizedInfo{
				UserID: "user123",
				TeamID: "team456",
				Scope:  "read",
			},
		},
		{
			name: "WithAuthorized from map - only constraints",
			input: map[string]interface{}{
				"user_id": "user123",
				"constraints": map[string]interface{}{
					"team_only": true,
					"extra": map[string]interface{}{
						"region": "us-west",
					},
				},
			},
			expected: &AuthorizedInfo{
				UserID: "user123",
				Constraints: DataConstraints{
					TeamOnly: true,
					Extra: map[string]interface{}{
						"region": "us-west",
					},
				},
			},
		},
		{
			name:     "WithAuthorized with nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			process := &Process{
				Name: "test.process",
				Args: []interface{}{},
			}

			result := process.WithAuthorized(tt.input)

			// Verify it returns the process for chaining
			if result != process {
				t.Errorf("WithAuthorized() should return the same process instance")
			}

			// If input is nil, authorized should remain nil
			if tt.input == nil {
				if process.Authorized != nil {
					t.Errorf("WithAuthorized(nil) should not set Authorized field")
				}
				return
			}

			// Verify the authorized info is set correctly
			if process.Authorized == nil {
				t.Fatalf("WithAuthorized() did not set Authorized field")
			}

			// Compare fields
			if process.Authorized.Subject != tt.expected.Subject {
				t.Errorf("Subject = %v, want %v", process.Authorized.Subject, tt.expected.Subject)
			}
			if process.Authorized.ClientID != tt.expected.ClientID {
				t.Errorf("ClientID = %v, want %v", process.Authorized.ClientID, tt.expected.ClientID)
			}
			if process.Authorized.Scope != tt.expected.Scope {
				t.Errorf("Scope = %v, want %v", process.Authorized.Scope, tt.expected.Scope)
			}
			if process.Authorized.SessionID != tt.expected.SessionID {
				t.Errorf("SessionID = %v, want %v", process.Authorized.SessionID, tt.expected.SessionID)
			}
			if process.Authorized.UserID != tt.expected.UserID {
				t.Errorf("UserID = %v, want %v", process.Authorized.UserID, tt.expected.UserID)
			}
			if process.Authorized.TeamID != tt.expected.TeamID {
				t.Errorf("TeamID = %v, want %v", process.Authorized.TeamID, tt.expected.TeamID)
			}
			if process.Authorized.TenantID != tt.expected.TenantID {
				t.Errorf("TenantID = %v, want %v", process.Authorized.TenantID, tt.expected.TenantID)
			}
			if process.Authorized.RememberMe != tt.expected.RememberMe {
				t.Errorf("RememberMe = %v, want %v", process.Authorized.RememberMe, tt.expected.RememberMe)
			}

			// Compare constraints
			if process.Authorized.Constraints.OwnerOnly != tt.expected.Constraints.OwnerOnly {
				t.Errorf("Constraints.OwnerOnly = %v, want %v", process.Authorized.Constraints.OwnerOnly, tt.expected.Constraints.OwnerOnly)
			}
			if process.Authorized.Constraints.CreatorOnly != tt.expected.Constraints.CreatorOnly {
				t.Errorf("Constraints.CreatorOnly = %v, want %v", process.Authorized.Constraints.CreatorOnly, tt.expected.Constraints.CreatorOnly)
			}
			if process.Authorized.Constraints.EditorOnly != tt.expected.Constraints.EditorOnly {
				t.Errorf("Constraints.EditorOnly = %v, want %v", process.Authorized.Constraints.EditorOnly, tt.expected.Constraints.EditorOnly)
			}
			if process.Authorized.Constraints.TeamOnly != tt.expected.Constraints.TeamOnly {
				t.Errorf("Constraints.TeamOnly = %v, want %v", process.Authorized.Constraints.TeamOnly, tt.expected.Constraints.TeamOnly)
			}

			// Compare Extra map
			if tt.expected.Constraints.Extra != nil {
				if process.Authorized.Constraints.Extra == nil {
					t.Errorf("Constraints.Extra should not be nil")
				} else {
					for key, expectedValue := range tt.expected.Constraints.Extra {
						actualValue, ok := process.Authorized.Constraints.Extra[key]
						if !ok {
							t.Errorf("Constraints.Extra missing key %v", key)
						} else if actualValue != expectedValue {
							t.Errorf("Constraints.Extra[%v] = %v, want %v", key, actualValue, expectedValue)
						}
					}
				}
			}
		})
	}
}

func TestProcess_WithAuthorized_Chaining(t *testing.T) {
	process := &Process{
		Name: "test.process",
		Args: []interface{}{},
	}

	// Test method chaining
	result := process.
		WithAuthorized(map[string]interface{}{
			"user_id": "user123",
			"team_id": "team456",
		}).
		WithSID("session123").
		WithGlobal(map[string]interface{}{"key": "value"})

	if result != process {
		t.Errorf("Method chaining should return the same process instance")
	}

	if process.Authorized == nil {
		t.Fatalf("Authorized should be set")
	}

	if process.Authorized.UserID != "user123" {
		t.Errorf("UserID = %v, want user123", process.Authorized.UserID)
	}

	if process.Authorized.TeamID != "team456" {
		t.Errorf("TeamID = %v, want team456", process.Authorized.TeamID)
	}

	if process.Sid != "session123" {
		t.Errorf("Sid = %v, want session123", process.Sid)
	}

	if process.Global["key"] != "value" {
		t.Errorf("Global[key] = %v, want value", process.Global["key"])
	}
}

func TestProcess_GetAuthorized(t *testing.T) {
	t.Run("GetAuthorized with existing info", func(t *testing.T) {
		authInfo := &AuthorizedInfo{
			UserID: "user123",
			TeamID: "team456",
		}

		process := &Process{
			Name:       "test.process",
			Authorized: authInfo,
		}

		result := process.GetAuthorized()
		if result == nil {
			t.Fatalf("GetAuthorized() should return authorized info")
		}

		if result.UserID != "user123" {
			t.Errorf("UserID = %v, want user123", result.UserID)
		}

		if result.TeamID != "team456" {
			t.Errorf("TeamID = %v, want team456", result.TeamID)
		}
	})

	t.Run("GetAuthorized with nil", func(t *testing.T) {
		process := &Process{
			Name:       "test.process",
			Authorized: nil,
		}

		result := process.GetAuthorized()
		if result == nil {
			t.Fatalf("GetAuthorized() should return empty AuthorizedInfo, not nil")
		}

		if result.UserID != "" {
			t.Errorf("UserID should be empty, got %v", result.UserID)
		}
	})
}

func TestAuthorizedInfo_AuthorizedToMap(t *testing.T) {
	tests := []struct {
		name     string
		auth     *AuthorizedInfo
		expected map[string]interface{}
	}{
		{
			name: "Full AuthorizedInfo",
			auth: &AuthorizedInfo{
				Subject:    "user123",
				ClientID:   "client456",
				Scope:      "read write",
				SessionID:  "session789",
				UserID:     "user123",
				TeamID:     "team456",
				TenantID:   "tenant789",
				RememberMe: true,
				Constraints: DataConstraints{
					OwnerOnly:   true,
					CreatorOnly: false,
					EditorOnly:  false,
					TeamOnly:    true,
					Extra: map[string]interface{}{
						"department": "engineering",
					},
				},
			},
			expected: map[string]interface{}{
				"sub":         "user123",
				"client_id":   "client456",
				"scope":       "read write",
				"session_id":  "session789",
				"user_id":     "user123",
				"team_id":     "team456",
				"tenant_id":   "tenant789",
				"remember_me": true,
				"constraints": map[string]interface{}{
					"owner_only": true,
					"team_only":  true,
					"extra": map[string]interface{}{
						"department": "engineering",
					},
				},
			},
		},
		{
			name: "Partial AuthorizedInfo",
			auth: &AuthorizedInfo{
				UserID: "user123",
				TeamID: "team456",
			},
			expected: map[string]interface{}{
				"user_id": "user123",
				"team_id": "team456",
			},
		},
		{
			name: "AuthorizedInfo with only constraints",
			auth: &AuthorizedInfo{
				UserID: "user123",
				Constraints: DataConstraints{
					TeamOnly: true,
					Extra: map[string]interface{}{
						"region": "us-west",
					},
				},
			},
			expected: map[string]interface{}{
				"user_id": "user123",
				"constraints": map[string]interface{}{
					"team_only": true,
					"extra": map[string]interface{}{
						"region": "us-west",
					},
				},
			},
		},
		{
			name:     "Nil AuthorizedInfo",
			auth:     nil,
			expected: nil,
		},
		{
			name:     "Empty AuthorizedInfo",
			auth:     &AuthorizedInfo{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.auth.AuthorizedToMap()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("AuthorizedToMap() should return nil for nil input")
				}
				return
			}

			if result == nil {
				t.Fatalf("AuthorizedToMap() returned nil, expected map")
			}

			// Check all expected keys
			for key, expectedValue := range tt.expected {
				actualValue, ok := result[key]
				if !ok {
					t.Errorf("Key %s not found in result", key)
					continue
				}

				// Special handling for nested maps (constraints)
				if key == "constraints" {
					expectedConstraints, _ := expectedValue.(map[string]interface{})
					actualConstraints, ok := actualValue.(map[string]interface{})
					if !ok {
						t.Errorf("constraints should be map[string]interface{}, got %T", actualValue)
						continue
					}

					for cKey, cExpectedValue := range expectedConstraints {
						cActualValue, ok := actualConstraints[cKey]
						if !ok {
							t.Errorf("Constraint key %s not found", cKey)
							continue
						}

						// Special handling for nested extra map
						if cKey == "extra" {
							expectedExtra, _ := cExpectedValue.(map[string]interface{})
							actualExtra, ok := cActualValue.(map[string]interface{})
							if !ok {
								t.Errorf("extra should be map[string]interface{}, got %T", cActualValue)
								continue
							}

							for eKey, eExpectedValue := range expectedExtra {
								eActualValue, ok := actualExtra[eKey]
								if !ok {
									t.Errorf("Extra key %s not found", eKey)
								} else if eActualValue != eExpectedValue {
									t.Errorf("Extra[%s] = %v, want %v", eKey, eActualValue, eExpectedValue)
								}
							}
						} else if cActualValue != cExpectedValue {
							t.Errorf("Constraint[%s] = %v, want %v", cKey, cActualValue, cExpectedValue)
						}
					}
				} else if actualValue != expectedValue {
					t.Errorf("Key %s = %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Check no unexpected keys (except for empty maps)
			if len(tt.expected) > 0 {
				for key := range result {
					if _, ok := tt.expected[key]; !ok {
						t.Errorf("Unexpected key %s in result", key)
					}
				}
			}
		})
	}
}
