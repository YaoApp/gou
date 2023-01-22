package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidationTypeof(t *testing.T) {
	assert.True(t, ValidationTypeof("foo", nil, "string"))
	assert.False(t, ValidationTypeof(1, nil, "string"))
	assert.False(t, ValidationTypeof(0.618, nil, "string"))

	assert.True(t, ValidationTypeof(1, nil, "integer"))
	assert.False(t, ValidationTypeof(0.618, nil, "integer"))
	assert.False(t, ValidationTypeof("foo", nil, "integer"))

	assert.True(t, ValidationTypeof(0.618, nil, "float"))
	assert.False(t, ValidationTypeof(1, nil, "float"))
	assert.False(t, ValidationTypeof("foo", nil, "float"))

	assert.True(t, ValidationTypeof(0.618, nil, "number"))
	assert.True(t, ValidationTypeof(1, nil, "number"))
	assert.False(t, ValidationTypeof("foo", nil, "number"))

	assert.True(t, ValidationTypeof("2021-08-20 22:22:33", nil, "datetime"))
	assert.True(t, ValidationTypeof("2021-08-20T22:22:33", nil, "datetime"))
	assert.False(t, ValidationTypeof("2021-08-20 22:22:33139", nil, "datetime"))
	assert.False(t, ValidationTypeof(1, nil, "datetime"))

	assert.True(t, ValidationTypeof(true, nil, "bool"))
	assert.True(t, ValidationTypeof(false, nil, "bool"))
	assert.True(t, ValidationTypeof(1, nil, "bool"))
	assert.True(t, ValidationTypeof(0, nil, "bool"))
	assert.False(t, ValidationTypeof("foo", nil, "bool"))
}

func TestValidationMin(t *testing.T) {
	assert.True(t, ValidationMin(100, nil, 100))
	assert.True(t, ValidationMin(101, nil, 100))
	assert.False(t, ValidationMin(99, nil, 100))

	assert.True(t, ValidationMin(100.00, nil, 100.00))
	assert.True(t, ValidationMin(101.00, nil, 100.00))
	assert.False(t, ValidationMin(99.00, nil, 100.00))

	assert.True(t, ValidationMin(100.00, nil, 100))
	assert.True(t, ValidationMin(101.00, nil, 100))
	assert.False(t, ValidationMin(99.00, nil, 100))
}

func TestValidationMax(t *testing.T) {
	assert.True(t, ValidationMax(100, nil, 100))
	assert.True(t, ValidationMax(99, nil, 100))
	assert.False(t, ValidationMax(101, nil, 100))

	assert.True(t, ValidationMax(100.00, nil, 100.00))
	assert.True(t, ValidationMax(99.00, nil, 100.00))
	assert.False(t, ValidationMax(101.00, nil, 100.00))

	assert.True(t, ValidationMax(100.00, nil, 100))
	assert.True(t, ValidationMax(99.00, nil, 100))
	assert.False(t, ValidationMax(101.00, nil, 100))
}

func TestValidationMinLength(t *testing.T) {
	assert.True(t, ValidationMinLength("foo", nil, 3))
	assert.True(t, ValidationMinLength("foobar", nil, 3))
	assert.False(t, ValidationMinLength("fo", nil, 3))
}

func TestValidationMaxLength(t *testing.T) {
	assert.True(t, ValidationMaxLength("foo", nil, 3))
	assert.True(t, ValidationMaxLength("fo", nil, 3))
	assert.False(t, ValidationMaxLength("foobar", nil, 3))
}

func TestValidationPattern(t *testing.T) {
	assert.True(t, ValidationPattern("1983", nil, "^[0-9]{4}$"))
	assert.True(t, ValidationPattern(1983, nil, "^[0-9]{4}$"))
	assert.False(t, ValidationPattern("83", nil, "^[0-9]{4}$"))
	assert.False(t, ValidationPattern(83, nil, "^[0-9]{4}$"))
	assert.False(t, ValidationPattern("nine", nil, "^[0-9]{4}$"))
}

func TestValidationEnum(t *testing.T) {
	assert.True(t, ValidationEnum("disabled", nil, "enabled", "disabled"))
	assert.False(t, ValidationEnum("notin", nil, "enabled", "disabled"))

	assert.True(t, ValidationEnum(1024, nil, 1024, 1983))
	assert.False(t, ValidationEnum(2077, nil, 1024, 1983))

	assert.True(t, ValidationEnum(1024.00, nil, 1024.00, 1983))
	assert.False(t, ValidationEnum(1024, nil, 1024.00, 1983))
	assert.False(t, ValidationEnum(2077, nil, 1024.00, 1983))

	assert.True(t, ValidationEnum(1024, nil, 1024, 1983, "7749"))
	assert.True(t, ValidationEnum("7749", nil, 1024, 1983, "7749"))
}

func TestValidationEmail(t *testing.T) {
	assert.True(t, ValidationEmail("xiang@iqka.com", nil))
	assert.False(t, ValidationEmail("xiang", nil))
	assert.False(t, ValidationEmail(1, nil))
}

func TestValidationMobile(t *testing.T) {
	assert.True(t, ValidationMobile("13299991111", nil))
	assert.True(t, ValidationMobile("123-123-1234", nil, "us"))
	assert.False(t, ValidationMobile("xiang", nil))
	assert.False(t, ValidationMobile(1, nil))
}
