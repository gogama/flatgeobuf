// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package flat

import "strconv"

type ColumnType byte

const (
	ColumnTypeByte     ColumnType = 0
	ColumnTypeUByte    ColumnType = 1
	ColumnTypeBool     ColumnType = 2
	ColumnTypeShort    ColumnType = 3
	ColumnTypeUShort   ColumnType = 4
	ColumnTypeInt      ColumnType = 5
	ColumnTypeUInt     ColumnType = 6
	ColumnTypeLong     ColumnType = 7
	ColumnTypeULong    ColumnType = 8
	ColumnTypeFloat    ColumnType = 9
	ColumnTypeDouble   ColumnType = 10
	ColumnTypeString   ColumnType = 11
	ColumnTypeJson     ColumnType = 12
	ColumnTypeDateTime ColumnType = 13
	ColumnTypeBinary   ColumnType = 14
)

var EnumNamesColumnType = map[ColumnType]string{
	ColumnTypeByte:     "Byte",
	ColumnTypeUByte:    "UByte",
	ColumnTypeBool:     "Bool",
	ColumnTypeShort:    "Short",
	ColumnTypeUShort:   "UShort",
	ColumnTypeInt:      "Int",
	ColumnTypeUInt:     "UInt",
	ColumnTypeLong:     "Long",
	ColumnTypeULong:    "ULong",
	ColumnTypeFloat:    "Float",
	ColumnTypeDouble:   "Double",
	ColumnTypeString:   "String",
	ColumnTypeJson:     "Json",
	ColumnTypeDateTime: "DateTime",
	ColumnTypeBinary:   "Binary",
}

var EnumValuesColumnType = map[string]ColumnType{
	"Byte":     ColumnTypeByte,
	"UByte":    ColumnTypeUByte,
	"Bool":     ColumnTypeBool,
	"Short":    ColumnTypeShort,
	"UShort":   ColumnTypeUShort,
	"Int":      ColumnTypeInt,
	"UInt":     ColumnTypeUInt,
	"Long":     ColumnTypeLong,
	"ULong":    ColumnTypeULong,
	"Float":    ColumnTypeFloat,
	"Double":   ColumnTypeDouble,
	"String":   ColumnTypeString,
	"Json":     ColumnTypeJson,
	"DateTime": ColumnTypeDateTime,
	"Binary":   ColumnTypeBinary,
}

func (v ColumnType) String() string {
	if s, ok := EnumNamesColumnType[v]; ok {
		return s
	}
	return "ColumnType(" + strconv.FormatInt(int64(v), 10) + ")"
}