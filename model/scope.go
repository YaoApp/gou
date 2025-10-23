package model

import "github.com/yaoapp/kun/maps"

// NewAccessScope creates a new AccessScope instance
// Supports both maps.MapStr and map[string]interface{}
func NewAccessScope(data ...interface{}) *AccessScope {
	scope := &AccessScope{}

	if len(data) == 0 {
		return scope
	}

	switch v := data[0].(type) {
	case maps.MapStr:
		if createdBy, ok := v["__yao_created_by"].(string); ok {
			scope.CreatedBy = createdBy
		}
		if updatedBy, ok := v["__yao_updated_by"].(string); ok {
			scope.UpdatedBy = updatedBy
		}
		if teamID, ok := v["__yao_team_id"].(string); ok {
			scope.TeamID = teamID
		}
		if tenantID, ok := v["__yao_tenant_id"].(string); ok {
			scope.TenantID = tenantID
		}

	case map[string]interface{}:
		if createdBy, ok := v["__yao_created_by"].(string); ok {
			scope.CreatedBy = createdBy
		}
		if updatedBy, ok := v["__yao_updated_by"].(string); ok {
			scope.UpdatedBy = updatedBy
		}
		if teamID, ok := v["__yao_team_id"].(string); ok {
			scope.TeamID = teamID
		}
		if tenantID, ok := v["__yao_tenant_id"].(string); ok {
			scope.TenantID = tenantID
		}
	}

	return scope
}

// Wheres converts AccessScope to QueryWhere conditions (includes all non-empty fields with AND logic)
func (scope *AccessScope) Wheres(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	if scope.CreatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	}

	if scope.UpdatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
		})
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresTeamOnly adds only team and tenant filters
func (scope *AccessScope) WheresTeamOnly(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresCreatorOnly adds only creator filter along with team/tenant
func (scope *AccessScope) WheresCreatorOnly(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	if scope.CreatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresEditorOnly adds only editor filter along with team/tenant
func (scope *AccessScope) WheresEditorOnly(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	if scope.UpdatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
		})
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresCreatorAndEditor adds both creator and editor filters (AND logic) along with team/tenant
func (scope *AccessScope) WheresCreatorAndEditor(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	if scope.CreatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	}

	if scope.UpdatedBy != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
		})
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresCreatorOrEditor adds creator or editor filter (OR logic) along with team/tenant
func (scope *AccessScope) WheresCreatorOrEditor(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	// Add OR group for creator or editor
	if scope.CreatedBy != "" && scope.UpdatedBy != "" {
		// Both creator and editor exist, use OR logic
		wheres = append(wheres, QueryWhere{
			Wheres: []QueryWhere{
				{
					Column: "__yao_created_by",
					Value:  scope.CreatedBy,
					OP:     "eq",
				},
				{
					Column: "__yao_updated_by",
					Value:  scope.UpdatedBy,
					OP:     "eq",
					Method: "orwhere",
				},
			},
		})
	} else if scope.CreatedBy != "" {
		// Only creator exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	} else if scope.UpdatedBy != "" {
		// Only editor exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
		})
	}

	if scope.TeamID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresTeamOrCreator adds team or creator filter (OR logic) along with tenant
func (scope *AccessScope) WheresTeamOrCreator(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	// Add OR group for team or creator
	if scope.TeamID != "" && scope.CreatedBy != "" {
		// Both team and creator exist, use OR logic
		wheres = append(wheres, QueryWhere{
			Wheres: []QueryWhere{
				{
					Column: "__yao_team_id",
					Value:  scope.TeamID,
					OP:     "eq",
				},
				{
					Column: "__yao_created_by",
					Value:  scope.CreatedBy,
					OP:     "eq",
					Method: "orwhere",
				},
			},
		})
	} else if scope.TeamID != "" {
		// Only team exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	} else if scope.CreatedBy != "" {
		// Only creator exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresTeamOrEditor adds team or editor filter (OR logic) along with tenant
func (scope *AccessScope) WheresTeamOrEditor(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	// Add OR group for team or editor
	if scope.TeamID != "" && scope.UpdatedBy != "" {
		// Both team and editor exist, use OR logic
		wheres = append(wheres, QueryWhere{
			Wheres: []QueryWhere{
				{
					Column: "__yao_team_id",
					Value:  scope.TeamID,
					OP:     "eq",
				},
				{
					Column: "__yao_updated_by",
					Value:  scope.UpdatedBy,
					OP:     "eq",
					Method: "orwhere",
				},
			},
		})
	} else if scope.TeamID != "" {
		// Only team exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
		})
	} else if scope.UpdatedBy != "" {
		// Only editor exists
		wheres = append(wheres, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// WheresCreatorOrTeamOrEditor adds creator, team, or editor filter (OR logic) along with tenant
func (scope *AccessScope) WheresCreatorOrTeamOrEditor(wheres []QueryWhere) []QueryWhere {
	if wheres == nil {
		wheres = []QueryWhere{}
	}

	// Build OR group with all available conditions
	orConditions := []QueryWhere{}

	if scope.CreatedBy != "" {
		orConditions = append(orConditions, QueryWhere{
			Column: "__yao_created_by",
			Value:  scope.CreatedBy,
			OP:     "eq",
		})
	}

	if scope.UpdatedBy != "" {
		orConditions = append(orConditions, QueryWhere{
			Column: "__yao_updated_by",
			Value:  scope.UpdatedBy,
			OP:     "eq",
			Method: "orwhere",
		})
	}

	if scope.TeamID != "" {
		orConditions = append(orConditions, QueryWhere{
			Column: "__yao_team_id",
			Value:  scope.TeamID,
			OP:     "eq",
			Method: "orwhere",
		})
	}

	// Add the OR group if there are conditions
	if len(orConditions) > 0 {
		wheres = append(wheres, QueryWhere{
			Wheres: orConditions,
		})
	}

	if scope.TenantID != "" {
		wheres = append(wheres, QueryWhere{
			Column: "__yao_tenant_id",
			Value:  scope.TenantID,
			OP:     "eq",
		})
	}

	return wheres
}

// Append appends AccessScope fields to data map for database insertion
// Supports both maps.MapStr and map[string]interface{}, returns map[string]interface{}
func (scope *AccessScope) Append(data interface{}) map[string]interface{} {
	result := map[string]interface{}{}

	// Convert input to map[string]interface{}
	switch v := data.(type) {
	case maps.MapStr:
		for key, value := range v {
			result[key] = value
		}
	case map[string]interface{}:
		for key, value := range v {
			result[key] = value
		}
	}

	// Append scope fields
	if scope.CreatedBy != "" {
		result["__yao_created_by"] = scope.CreatedBy
	}
	if scope.UpdatedBy != "" {
		result["__yao_updated_by"] = scope.UpdatedBy
	}
	if scope.TeamID != "" {
		result["__yao_team_id"] = scope.TeamID
	}
	if scope.TenantID != "" {
		result["__yao_tenant_id"] = scope.TenantID
	}

	return result
}
