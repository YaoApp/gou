package model

import (
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/day"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/str"
	"github.com/yaoapp/xun/dbal"
	"github.com/yaoapp/xun/dbal/schema"
)

// FliterIn 输入过滤器
func (column *Column) FliterIn(value interface{}, row maps.MapStrAny) {
	column.fliterInCrypt(value, row)
	column.fliterInJSON(value, row)
	column.fliterInDateTime(value, row)
}

// FliterOut 输出过滤器
func (column *Column) FliterOut(value interface{}, row maps.MapStrAny, export ...string) {
	exportName := ""
	if len(export) > 0 {
		exportName = export[0]
	}
	column.fliterOutJSON(value, row, exportName)
}

// fliterInJSON JSON字段处理
func (column *Column) fliterOutJSON(value interface{}, row maps.MapStrAny, export string) {
	if strings.ToLower(column.Type) != "json" {
		return
	}
	name := column.Name
	if export != "" {
		name = export
	}

	if raw, ok := value.(string); ok {

		var v interface{}
		err := jsoniter.UnmarshalFromString(raw, &v)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		row.Set(name, v)
	} else if raw, ok := value.([]byte); ok {
		var v interface{}
		err := jsoniter.Unmarshal(raw, &v)
		if err != nil {
			exception.Err(err, 400).Throw()
		}
		row.Set(name, v)
	}
}

// fliterInCrypt 加密字段处理
func (column *Column) fliterInCrypt(value interface{}, row maps.MapStrAny) {
	if column.Crypt == "" {
		return
	}

	// 忽略空值
	if value == nil {
		return
	}

	icrypt, err := SelectCrypt(column.Crypt)
	if err != nil {
		exception.New(err.Error(), 400).Throw()
	}

	valuestr, ok := value.(string)
	if !ok {
		exception.New(column.Name+"数值格式不是字符型", 400).Throw()
	}

	// 忽略除 MySQL 之外的 AES 驱动
	if column.Crypt == "AES" && column.model.Driver != "mysql" {
		column.Crypt = ""
		return
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

// fliterInDate 日期字段处理
func (column *Column) fliterInDateTime(value interface{}, row maps.MapStrAny) {
	typ := strings.ToLower(column.Type)
	switch typ {
	case "datetime", "date", "datetimeTz", "timestamp", "timestampTz", "time", "timeTz":
		if _, ok := value.(dbal.Expression); !ok {
			if value != nil {
				row.Set(column.Name, day.Of(value).Format("2006-01-02 15:04:05"))
			}
		}
	}
}

// fliterInJSON JSON字段处理
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
func (column *Column) Validate(value interface{}, row maps.MapStrAny) (bool, []string) {
	messages := []string{}
	success := true
	for _, v := range column.Validations {
		method, has := Validations[v.Method]
		if !has {
			continue
		}
		if !method(value, row, v.Args...) {
			data := column.Map()
			data["input"] = value
			message := str.Bind(v.Message, data)
			messages = append(messages, message)
			success = false
		}
	}
	return success, messages
}

// Map 转换为Map
func (column *Column) Map() map[string]interface{} {
	res := map[string]interface{}{}
	bytes, err := jsoniter.Marshal(column)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	err = jsoniter.Unmarshal(bytes, &res)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
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
	case "Boolean", "boolean":
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

	exception.New("类型错误 %s %s", 400, column.Type, column.Name).Throw()

	return nil
}
