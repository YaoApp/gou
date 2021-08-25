package gou

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/xun/dbal"
	"github.com/yaoapp/xun/dbal/schema"
)

// FliterIn 输入过滤器
func (column *Column) FliterIn(value interface{}, row maps.MapStrAny) {
	column.fliterInCrypt(value, row)
	column.fliterInJSON(value, row)
	column.Validate(value, row)
}

// fliterInCrypt 加密字段
func (column *Column) fliterInCrypt(value interface{}, row maps.MapStrAny) {
	if column.Crypt == "" {
		return
	}

	icrypt := SelectCrypt(column.Crypt)

	valuestr, ok := value.(string)
	if !ok {
		exception.New(column.Name+"数值格式不是字符型", 400).Throw()
	}

	if column.Crypt == "AES" {
		exp, err := icrypt.Encode(valuestr)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		row.Set(column.Name, dbal.Raw(exp))
		return
	}

	valuehash, err := icrypt.Encode(valuestr)
	if err != nil {
		exception.Err(err, 400).Throw()
	}

	row.Set(column.Name, valuehash)
}

// fliterInJSON JSON字段
func (column *Column) fliterInJSON(value interface{}, row maps.MapStrAny) {
	if strings.ToLower(column.Type) != "json" {
		return
	}
	bytes, err := jsoniter.Marshal(value)
	if err != nil {
		exception.Err(err, 400).Throw()
	}
	row.Set(column.Name, string(bytes))
}

// Validate 数值有效性验证
func (column *Column) Validate(value interface{}, row maps.MapStrAny) {
}

// SetOption 设置字段选项
func (column Column) SetOption(col *schema.Column) {
	if column.Comment != "" { // 注释
		col.SetComment(column.Comment)
	}

	if column.Default != nil { // 默认值
		col.SetDefault(column.Default)
	}

	if column.Nullable { // 可以为空
		col.Null()
	}

	if column.Index { // 索引
		col.Index()
	}

	if column.Unique { // 唯一索引
		col.Unique()
	}

	if column.Primary { // 主键
		col.Primary()
	}
}

// SetType 设置字段类型
func (column Column) SetType(table schema.Blueprint) *schema.Column {

	switch column.Type {
	// String
	case "string":
		return table.String(column.Name, column.Length)
	case "char":
		return table.Char(column.Name, column.Length)

	case "text":
		return table.Text(column.Name)
	case "mediumText":
		return table.MediumText(column.Name)
	case "longText":
		return table.LongText(column.Name)

	// Binary
	case "binary":
		return table.Binary(column.Name, column.Length)

	// Datetime
	case "date":
		return table.Date(column.Name)
	case "datetime":
		if column.Length > 0 {
			return table.DateTime(column.Name, column.Length)
		}
		return table.DateTime(column.Name)
	case "datetimeTz":
		if column.Length > 0 {
			return table.DateTime(column.Name, column.Length)

		}
		return table.DateTime(column.Name)
	case "time":
		if column.Length > 0 {
			return table.Time(column.Name, column.Length)

		}
		return table.Time(column.Name)

	case "timeTz":
		if column.Length > 0 {
			return table.TimeTz(column.Name, column.Length)

		}
		return table.TimeTz(column.Name)

	case "timestamp":
		if column.Length > 0 {
			return table.Timestamp(column.Name, column.Length)

		}
		return table.Timestamp(column.Name)

	case "timestampTz":
		if column.Length > 0 {
			return table.TimestampTz(column.Name, column.Length)

		}
		return table.TimestampTz(column.Name)

	// Numberic: Integer
	case "tinyInteger":
		return table.TinyInteger(column.Name)

	case "unsignedTinyInteger":
		return table.UnsignedTinyInteger(column.Name)

	case "tinyIncrements":
		return table.TinyIncrements(column.Name)

	case "smallInteger":
		return table.SmallInteger(column.Name)

	case "unsignedSmallInteger":
		return table.UnsignedSmallInteger(column.Name)

	case "smallIncrements":
		return table.SmallIncrements(column.Name)

	case "integer":
		return table.Integer(column.Name)

	case "unsignedInteger":
		return table.UnsignedInteger(column.Name)

	case "increments":
		return table.Increments(column.Name)

	case "bigInteger":
		return table.BigInteger(column.Name)

	case "unsignedBigInteger":
		return table.UnsignedBigInteger(column.Name)

	case "bigIncrements":
		return table.BigIncrements(column.Name)

	case "id", "ID":
		return table.ID(column.Name)

	// Numberic: Decimal
	case "decimal":
		return table.Decimal(column.Name, column.Precision, column.Scale)

	case "unsignedDecimal":
		return table.UnsignedDecimal(column.Name, column.Precision, column.Scale)

	case "float":
		return table.Float(column.Name, column.Precision, column.Scale)

	case "unsignedFloat":
		return table.UnsignedFloat(column.Name, column.Precision, column.Scale)

	case "double":
		return table.Double(column.Name, column.Precision, column.Scale)

	case "unsignedDouble":
		return table.UnsignedDouble(column.Name, column.Precision, column.Scale)

	// Boolen,enum
	case "Boolean":
		return table.Boolean(column.Name)

	case "enum":
		return table.Enum(column.Name, column.Option)

	// JSON
	case "json", "JSON":
		return table.JSON(column.Name)

	case "jsonb", "JSONB":
		return table.JSONB(column.Name)

	// uuid, ipAddress, macAddress, year etc.
	case "uuid":
		return table.UUID(column.Name)

	case "ipAddress":
		return table.IPAddress(column.Name)

	case "macAddress":
		return table.MACAddress(column.Name)

	case "year":
		return table.Year(column.Name)

	}

	return nil
}
