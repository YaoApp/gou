package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

func TestNewAccessScope(t *testing.T) {
	// Test empty scope
	scope := NewAccessScope()
	assert.NotNil(t, scope)
	assert.Empty(t, scope.CreatedBy)
	assert.Empty(t, scope.UpdatedBy)
	assert.Empty(t, scope.TeamID)
	assert.Empty(t, scope.TenantID)

	// Test with maps.MapStr
	data1 := maps.MapStr{
		"__yao_created_by": "user123",
		"__yao_updated_by": "user456",
		"__yao_team_id":    "team789",
		"__yao_tenant_id":  "tenant000",
	}
	scope1 := NewAccessScope(data1)
	assert.Equal(t, "user123", scope1.CreatedBy)
	assert.Equal(t, "user456", scope1.UpdatedBy)
	assert.Equal(t, "team789", scope1.TeamID)
	assert.Equal(t, "tenant000", scope1.TenantID)

	// Test with map[string]interface{}
	data2 := map[string]interface{}{
		"__yao_created_by": "user111",
		"__yao_team_id":    "team222",
	}
	scope2 := NewAccessScope(data2)
	assert.Equal(t, "user111", scope2.CreatedBy)
	assert.Empty(t, scope2.UpdatedBy)
	assert.Equal(t, "team222", scope2.TeamID)
	assert.Empty(t, scope2.TenantID)
}

func TestAccessScopeWheres(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	// Test with nil wheres
	wheres := scope.Wheres(nil)
	assert.NotNil(t, wheres)
	assert.Equal(t, 3, len(wheres))

	// Test with existing wheres
	existingWheres := []QueryWhere{
		{Column: "status", Value: "active", OP: "eq"},
	}
	wheres = scope.Wheres(existingWheres)
	assert.Equal(t, 4, len(wheres))
	assert.Equal(t, "status", wheres[0].Column)
	assert.Equal(t, "__yao_created_by", wheres[1].Column)
	assert.Equal(t, "user123", wheres[1].Value)
	assert.Equal(t, "__yao_team_id", wheres[2].Column)
	assert.Equal(t, "team456", wheres[2].Value)
	assert.Equal(t, "__yao_tenant_id", wheres[3].Column)
	assert.Equal(t, "tenant789", wheres[3].Value)

	// Test with empty scope
	emptyScope := &AccessScope{}
	wheres = emptyScope.Wheres(nil)
	assert.Equal(t, 0, len(wheres))
}

func TestAccessScopeAppend(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		UpdatedBy: "user456",
		TeamID:    "team789",
		TenantID:  "tenant000",
	}

	// Test with maps.MapStr
	data1 := maps.MapStr{"name": "test"}
	result1 := scope.Append(data1)
	assert.NotNil(t, result1)
	assert.Equal(t, "test", result1["name"])
	assert.Equal(t, "user123", result1["__yao_created_by"])
	assert.Equal(t, "user456", result1["__yao_updated_by"])
	assert.Equal(t, "team789", result1["__yao_team_id"])
	assert.Equal(t, "tenant000", result1["__yao_tenant_id"])

	// Test with map[string]interface{}
	data2 := map[string]interface{}{"name": "test2"}
	result2 := scope.Append(data2)
	assert.NotNil(t, result2)
	assert.Equal(t, "test2", result2["name"])
	assert.Equal(t, "user123", result2["__yao_created_by"])
	assert.Equal(t, "user456", result2["__yao_updated_by"])
	assert.Equal(t, "team789", result2["__yao_team_id"])
	assert.Equal(t, "tenant000", result2["__yao_tenant_id"])

	// Test with nil maps.MapStr
	var nilMapStr maps.MapStr
	result3 := scope.Append(nilMapStr)
	assert.NotNil(t, result3)
	assert.Equal(t, "user123", result3["__yao_created_by"])

	// Test with nil map[string]interface{}
	var nilMap map[string]interface{}
	result4 := scope.Append(nilMap)
	assert.NotNil(t, result4)
	assert.Equal(t, "user123", result4["__yao_created_by"])

	// Test with empty scope
	emptyScope := &AccessScope{}
	data3 := maps.MapStr{"name": "test3"}
	result5 := emptyScope.Append(data3)
	assert.NotNil(t, result5)
	assert.Equal(t, "test3", result5["name"])
	assert.Nil(t, result5["__yao_created_by"])
	assert.Nil(t, result5["__yao_updated_by"])

	// Test with partial scope
	partialScope := &AccessScope{
		TenantID: "tenant123",
	}
	data4 := maps.MapStr{"name": "test4"}
	result6 := partialScope.Append(data4)
	assert.NotNil(t, result6)
	assert.Equal(t, "test4", result6["name"])
	assert.Equal(t, "tenant123", result6["__yao_tenant_id"])
	assert.Nil(t, result6["__yao_created_by"])
}

func TestAccessScopeIntegration(t *testing.T) {
	// Test full workflow
	inputData := maps.MapStr{
		"__yao_created_by": "user123",
		"__yao_team_id":    "team456",
		"__yao_tenant_id":  "tenant789",
	}

	// Create scope from data
	scope := NewAccessScope(inputData)
	assert.Equal(t, "user123", scope.CreatedBy)
	assert.Equal(t, "team456", scope.TeamID)
	assert.Equal(t, "tenant789", scope.TenantID)

	// Convert to query conditions
	wheres := scope.Wheres(nil)
	assert.Equal(t, 3, len(wheres))

	// Append to data for insertion
	insertData := maps.MapStr{"name": "Test Record"}
	insertResult := scope.Append(insertData)
	assert.Equal(t, "Test Record", insertResult["name"])
	assert.Equal(t, "user123", insertResult["__yao_created_by"])
	assert.Equal(t, "team456", insertResult["__yao_team_id"])
	assert.Equal(t, "tenant789", insertResult["__yao_tenant_id"])

	// Update scenario - change updated_by
	scope.UpdatedBy = "user999"
	updateData := maps.MapStr{"name": "Updated Record"}
	updateResult := scope.Append(updateData)
	assert.Equal(t, "Updated Record", updateResult["name"])
	assert.Equal(t, "user999", updateResult["__yao_updated_by"])
}

func TestAccessScopeWithCount(t *testing.T) {
	// Test AccessScope with Count method
	scope := &AccessScope{
		TenantID: "tenant123",
		TeamID:   "team456",
	}

	// Create QueryParam with scope conditions
	param := QueryParam{}
	param.Wheres = scope.Wheres(param.Wheres)

	// Verify wheres are properly set
	assert.Equal(t, 2, len(param.Wheres))
	assert.Equal(t, "__yao_team_id", param.Wheres[0].Column)
	assert.Equal(t, "team456", param.Wheres[0].Value)
	assert.Equal(t, "__yao_tenant_id", param.Wheres[1].Column)
	assert.Equal(t, "tenant123", param.Wheres[1].Value)

	// Test with additional conditions
	scope2 := &AccessScope{
		CreatedBy: "user789",
	}
	param2 := QueryParam{
		Wheres: []QueryWhere{
			{Column: "status", Value: "active", OP: "eq"},
		},
	}
	param2.Wheres = scope2.Wheres(param2.Wheres)
	assert.Equal(t, 2, len(param2.Wheres))
	assert.Equal(t, "status", param2.Wheres[0].Column)
	assert.Equal(t, "__yao_created_by", param2.Wheres[1].Column)
}

func TestWheresTeamOnly(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresTeamOnly(nil)
	assert.Equal(t, 2, len(wheres))
	assert.Equal(t, "__yao_team_id", wheres[0].Column)
	assert.Equal(t, "team456", wheres[0].Value)
	assert.Equal(t, "__yao_tenant_id", wheres[1].Column)
	assert.Equal(t, "tenant789", wheres[1].Value)
}

func TestWheresCreatorOnly(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresCreatorOnly(nil)
	assert.Equal(t, 3, len(wheres))
	assert.Equal(t, "__yao_created_by", wheres[0].Column)
	assert.Equal(t, "user123", wheres[0].Value)
	assert.Equal(t, "__yao_team_id", wheres[1].Column)
	assert.Equal(t, "__yao_tenant_id", wheres[2].Column)
}

func TestWheresEditorOnly(t *testing.T) {
	scope := &AccessScope{
		UpdatedBy: "user456",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresEditorOnly(nil)
	assert.Equal(t, 3, len(wheres))
	assert.Equal(t, "__yao_updated_by", wheres[0].Column)
	assert.Equal(t, "user456", wheres[0].Value)
	assert.Equal(t, "__yao_team_id", wheres[1].Column)
	assert.Equal(t, "__yao_tenant_id", wheres[2].Column)
}

func TestWheresCreatorAndEditor(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		UpdatedBy: "user456",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresCreatorAndEditor(nil)
	assert.Equal(t, 4, len(wheres))
	assert.Equal(t, "__yao_created_by", wheres[0].Column)
	assert.Equal(t, "user123", wheres[0].Value)
	assert.Equal(t, "__yao_updated_by", wheres[1].Column)
	assert.Equal(t, "user456", wheres[1].Value)
}

func TestWheresCreatorOrEditor(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		UpdatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresCreatorOrEditor(nil)
	assert.Equal(t, 3, len(wheres))

	// First should be the OR group
	assert.NotNil(t, wheres[0].Wheres)
	assert.Equal(t, 2, len(wheres[0].Wheres))
	assert.Equal(t, "__yao_created_by", wheres[0].Wheres[0].Column)
	assert.Equal(t, "user123", wheres[0].Wheres[0].Value)
	assert.Equal(t, "__yao_updated_by", wheres[0].Wheres[1].Column)
	assert.Equal(t, "user123", wheres[0].Wheres[1].Value)
	assert.Equal(t, "orwhere", wheres[0].Wheres[1].Method)

	// Then team and tenant
	assert.Equal(t, "__yao_team_id", wheres[1].Column)
	assert.Equal(t, "__yao_tenant_id", wheres[2].Column)
}

func TestWheresCreatorOrEditorSingleField(t *testing.T) {
	// Test with only creator
	scope1 := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres1 := scope1.WheresCreatorOrEditor(nil)
	assert.Equal(t, 3, len(wheres1))
	assert.Equal(t, "__yao_created_by", wheres1[0].Column)
	assert.Equal(t, "user123", wheres1[0].Value)

	// Test with only editor
	scope2 := &AccessScope{
		UpdatedBy: "user456",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres2 := scope2.WheresCreatorOrEditor(nil)
	assert.Equal(t, 3, len(wheres2))
	assert.Equal(t, "__yao_updated_by", wheres2[0].Column)
	assert.Equal(t, "user456", wheres2[0].Value)
}

func TestWheresTeamOrCreator(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresTeamOrCreator(nil)
	assert.Equal(t, 2, len(wheres))

	// First should be the OR group
	assert.NotNil(t, wheres[0].Wheres)
	assert.Equal(t, 2, len(wheres[0].Wheres))
	assert.Equal(t, "__yao_team_id", wheres[0].Wheres[0].Column)
	assert.Equal(t, "team456", wheres[0].Wheres[0].Value)
	assert.Equal(t, "__yao_created_by", wheres[0].Wheres[1].Column)
	assert.Equal(t, "user123", wheres[0].Wheres[1].Value)
	assert.Equal(t, "orwhere", wheres[0].Wheres[1].Method)

	// Then tenant
	assert.Equal(t, "__yao_tenant_id", wheres[1].Column)
	assert.Equal(t, "tenant789", wheres[1].Value)
}

func TestWheresTeamOrEditor(t *testing.T) {
	scope := &AccessScope{
		UpdatedBy: "user456",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresTeamOrEditor(nil)
	assert.Equal(t, 2, len(wheres))

	// First should be the OR group
	assert.NotNil(t, wheres[0].Wheres)
	assert.Equal(t, 2, len(wheres[0].Wheres))
	assert.Equal(t, "__yao_team_id", wheres[0].Wheres[0].Column)
	assert.Equal(t, "team456", wheres[0].Wheres[0].Value)
	assert.Equal(t, "__yao_updated_by", wheres[0].Wheres[1].Column)
	assert.Equal(t, "user456", wheres[0].Wheres[1].Value)
	assert.Equal(t, "orwhere", wheres[0].Wheres[1].Method)

	// Then tenant
	assert.Equal(t, "__yao_tenant_id", wheres[1].Column)
}

func TestWheresCreatorOrTeamOrEditor(t *testing.T) {
	scope := &AccessScope{
		CreatedBy: "user123",
		UpdatedBy: "user456",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres := scope.WheresCreatorOrTeamOrEditor(nil)
	assert.Equal(t, 2, len(wheres))

	// First should be the OR group with all three conditions
	assert.NotNil(t, wheres[0].Wheres)
	assert.Equal(t, 3, len(wheres[0].Wheres))
	assert.Equal(t, "__yao_created_by", wheres[0].Wheres[0].Column)
	assert.Equal(t, "user123", wheres[0].Wheres[0].Value)
	assert.Equal(t, "__yao_updated_by", wheres[0].Wheres[1].Column)
	assert.Equal(t, "user456", wheres[0].Wheres[1].Value)
	assert.Equal(t, "orwhere", wheres[0].Wheres[1].Method)
	assert.Equal(t, "__yao_team_id", wheres[0].Wheres[2].Column)
	assert.Equal(t, "team456", wheres[0].Wheres[2].Value)
	assert.Equal(t, "orwhere", wheres[0].Wheres[2].Method)

	// Then tenant
	assert.Equal(t, "__yao_tenant_id", wheres[1].Column)
	assert.Equal(t, "tenant789", wheres[1].Value)
}

func TestWheresCreatorOrTeamOrEditorPartial(t *testing.T) {
	// Test with only creator
	scope1 := &AccessScope{
		CreatedBy: "user123",
		TenantID:  "tenant789",
	}

	wheres1 := scope1.WheresCreatorOrTeamOrEditor(nil)
	assert.Equal(t, 2, len(wheres1))
	assert.NotNil(t, wheres1[0].Wheres)
	assert.Equal(t, 1, len(wheres1[0].Wheres))
	assert.Equal(t, "__yao_created_by", wheres1[0].Wheres[0].Column)

	// Test with only team
	scope2 := &AccessScope{
		TeamID:   "team456",
		TenantID: "tenant789",
	}

	wheres2 := scope2.WheresCreatorOrTeamOrEditor(nil)
	assert.Equal(t, 2, len(wheres2))
	assert.NotNil(t, wheres2[0].Wheres)
	assert.Equal(t, 1, len(wheres2[0].Wheres))
	assert.Equal(t, "__yao_team_id", wheres2[0].Wheres[0].Column)

	// Test with creator and team
	scope3 := &AccessScope{
		CreatedBy: "user123",
		TeamID:    "team456",
		TenantID:  "tenant789",
	}

	wheres3 := scope3.WheresCreatorOrTeamOrEditor(nil)
	assert.Equal(t, 2, len(wheres3))
	assert.NotNil(t, wheres3[0].Wheres)
	assert.Equal(t, 2, len(wheres3[0].Wheres))
}
