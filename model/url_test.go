package model

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryUrlValues(t *testing.T) {
	params := url.Values{}
	params.Add("select", "name,secret,status,type")
	params.Add("with", "mother,addresses")
	params.Add("mother.select", "name,mobile,type,status")
	params.Add("addresses.select", "province,city,location,status")
	params.Add("where.status.eq", "enabled")
	params.Add("where.secret.notnull", "")
	params.Add("where.resume.null", "")
	params.Add("where.mobile.eq", "13900002222")
	params.Add("orwhere.mobile.eq", "13900001111")
	params.Add("where.mother.friends.status.eq", "enabled")
	params.Add("group.types.where.type.eq", "admin")
	params.Add("group.types.orwhere.type.eq", "staff")
	params.Add("order", "id.desc,name")
	param := URLToQueryParam(params)
	assert.Equal(t, param.Select, []interface{}{"name", "secret", "status", "type"})
	assert.Equal(t, len(param.Wheres), 7)
	assert.Equal(t, len(param.Withs), 2)
	assert.Equal(t, len(param.Orders), 2)
}
