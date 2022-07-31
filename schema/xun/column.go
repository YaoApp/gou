package xun

import (
	"fmt"

	"github.com/yaoapp/gou/schema/types"
	"github.com/yaoapp/xun/dbal/schema"
)

// setColumn
func setColumn(table schema.Blueprint, column types.Column) (*schema.Column, error) {

	col, err := setColumnType(table, column)
	if err != nil {
		return nil, err
	}

	if column.Comment != "" {
		col.SetComment(column.Comment)
	}

	if column.Default != nil {
		col.SetDefault(column.Default)
	}

	if column.Nullable {
		col.Null()
	}

	if column.Index {
		col.Index()
	}

	if column.Unique {
		col.Unique()
	}

	if column.Primary {
		col.Primary()
	}

	return col, nil
}

func setColumnType(table schema.Blueprint, column types.Column) (*schema.Column, error) {
	if column.Name == "" {
		return nil, fmt.Errorf("missing name %v", column)
	}

	switch column.Type {
	// String
	case "string":
		return table.String(column.Name, column.Length), nil
	case "char":
		return table.Char(column.Name, column.Length), nil

	case "text":
		return table.Text(column.Name), nil
	case "mediumText":
		return table.MediumText(column.Name), nil
	case "longText":
		return table.LongText(column.Name), nil

	// Binary
	case "binary":
		return table.Binary(column.Name, column.Length), nil

	// Datetime
	case "date":
		return table.Date(column.Name), nil
	case "datetime":
		if column.Length > 0 {
			return table.DateTime(column.Name, column.Length), nil
		}
		return table.DateTime(column.Name), nil
	case "datetimeTz":
		if column.Length > 0 {
			return table.DateTime(column.Name, column.Length), nil

		}
		return table.DateTime(column.Name), nil
	case "time":
		if column.Length > 0 {
			return table.Time(column.Name, column.Length), nil

		}
		return table.Time(column.Name), nil
	case "timeTz":
		if column.Length > 0 {
			return table.TimeTz(column.Name, column.Length), nil

		}
		return table.TimeTz(column.Name), nil

	case "timestamp":
		if column.Length > 0 {
			return table.Timestamp(column.Name, column.Length), nil

		}
		return table.Timestamp(column.Name), nil

	case "timestampTz":
		if column.Length > 0 {
			return table.TimestampTz(column.Name, column.Length), nil

		}
		return table.TimestampTz(column.Name), nil

	// Numberic: Integer
	case "tinyInteger":
		return table.TinyInteger(column.Name), nil

	case "unsignedTinyInteger":
		return table.UnsignedTinyInteger(column.Name), nil

	case "tinyIncrements":
		return table.TinyIncrements(column.Name), nil

	case "smallInteger":
		return table.SmallInteger(column.Name), nil

	case "unsignedSmallInteger":
		return table.UnsignedSmallInteger(column.Name), nil

	case "smallIncrements":
		return table.SmallIncrements(column.Name), nil

	case "integer":
		return table.Integer(column.Name), nil

	case "unsignedInteger":
		return table.UnsignedInteger(column.Name), nil

	case "increments":
		return table.Increments(column.Name), nil

	case "bigInteger":
		return table.BigInteger(column.Name), nil

	case "unsignedBigInteger":
		return table.UnsignedBigInteger(column.Name), nil

	case "bigIncrements":
		return table.BigIncrements(column.Name), nil

	case "id", "ID":
		return table.ID(column.Name), nil

	// Numberic: Decimal
	case "decimal":
		return table.Decimal(column.Name, column.Precision, column.Scale), nil

	case "unsignedDecimal":
		return table.UnsignedDecimal(column.Name, column.Precision, column.Scale), nil

	case "float":
		return table.Float(column.Name, column.Precision, column.Scale), nil

	case "unsignedFloat":
		return table.UnsignedFloat(column.Name, column.Precision, column.Scale), nil

	case "double":
		return table.Double(column.Name, column.Precision, column.Scale), nil

	case "unsignedDouble":
		return table.UnsignedDouble(column.Name, column.Precision, column.Scale), nil

	// Boolen,enum
	case "Boolean", "boolean":
		return table.Boolean(column.Name), nil

	case "enum":
		return table.Enum(column.Name, column.Option), nil

	// JSON
	case "json", "JSON":
		return table.JSON(column.Name), nil

	case "jsonb", "JSONB":
		return table.JSONB(column.Name), nil

	// uuid, ipAddress, macAddress, year etc.
	case "uuid":
		return table.UUID(column.Name), nil

	case "ipAddress":
		return table.IPAddress(column.Name), nil

	case "macAddress":
		return table.MACAddress(column.Name), nil

	case "year":
		return table.Year(column.Name), nil
	}

	return nil, fmt.Errorf("Column %s, Type %s does support", column.Name, column.Type)
}
