package gou

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// NewImport 创建导入器
func NewImport(chunk int) *Import {
	return &Import{
		Chunk:  chunk,
		Notify: func(line int, record string) {},
		Option: map[string]interface{}{},
	}
}

// Using 配置源数据与数据模型字段映射表或处理器
func (impt *Import) Using(process interface{}) *Import {
	if process, ok := process.(map[string]string); ok {
		impt.Mapping = process
	}

	if process, ok := process.(string); ok {
		impt.Process = process
	}
	return impt
}

// Set 配置源数据与数据模型字段映射表或处理器
func (impt *Import) Set(key string, value interface{}) *Import {
	impt.Option[key] = value
	return impt
}

// NewResponse 创建导入器
func (impt *Import) NewResponse() *ResponseImport {
	return &ResponseImport{
		Errors: &ErrorsImport{},
		Import: impt,
	}
}

// InsertCSV 导入CSV使用 (insert)
func (impt *Import) InsertCSV(csvfile string, model string) *ResponseImport {

	mod := Select(model)

	resp := impt.NewResponse()
	f, err := os.Open(csvfile)
	defer f.Close()
	if err != nil {
		resp.Error("打开文件错误, %s", 400, err.Error()).FileIs(csvfile)
		return resp
	}

	buf := bufio.NewReader(f)
	reader := csv.NewReader(buf)

	// CSV 分割符
	if comma, has := impt.Option["comma"]; has {
		if comma, ok := comma.(string); ok {
			reader.Comma = []rune(comma)[0]
		}
	}

	// CSV 换行符
	if comment, has := impt.Option["comment"]; has {
		if comment, ok := comment.(string); ok {
			reader.Comma = []rune(comment)[0]
		}
	}

	// CSV 去掉前导空格
	reader.TrimLeadingSpace = true
	if trimLeadingSpace, has := impt.Option["trim"]; has {
		if trimLeadingSpace, ok := trimLeadingSpace.(bool); ok {
			reader.TrimLeadingSpace = trimLeadingSpace
		}
	}

	// CSV 记录位置
	reader.FieldsPerRecord = 0
	if fieldsPerRecord, has := impt.Option["fields"]; has {
		if fieldsPerRecord, ok := fieldsPerRecord.(int); ok {
			reader.FieldsPerRecord = fieldsPerRecord
		}
	}

	// 遍历数据
	line := 0
	isColumn := true
	fields := []string{}
	data := []map[string]string{}
	for {
		line++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if isColumn { // 字段清单
			fields = record
			isColumn = false
			continue
		}

		if len(record) < len(fields) {
			resp.Error("缺少字段", 400).FileIs(csvfile).AtLine(line)
			continue
		}

		row := map[string]string{}
		for i := range fields {
			key := fields[i]
			value := record[i]
			row[key] = value
		}
		data = append(data, row)
		size := len(data)
		if size == impt.Chunk {
			resp.InsertChunk(line-size, data, mod)
			data = []map[string]string{}
		}
	}

	// 插入最后一批数据
	size := len(data)
	if size >= 0 {
		resp.InsertChunk(line-size, data, mod)
	}

	return resp
}

// InsertChunk 导入数据 (insert)
func (resp *ResponseImport) InsertChunk(line int, data []map[string]string, model *Model) {
	total := len(data)
	resp.Total = resp.Total + total
	columns, values := resp.CastInsert(line, data, model)
	err := model.Insert(columns, values)
	if err != nil {
		resp.Failure = resp.Failure + total
		resp.Error(err.Error(), 500).AtLine(line).ValueIs(data)
		return
	}
	resp.Success = resp.Success + total
}

// CastInsert 数据格式转换
func (resp *ResponseImport) CastInsert(line int, data []map[string]string, model *Model) ([]string, [][]interface{}) {

	records := []map[string]interface{}{}
	if resp.Import.Mapping != nil {
		records = resp.castByMapping(line, data, model)
	} else if resp.Import.Process != "" {
		records = resp.castByProcess(line, data, model)
	} else {
		records = resp.castType(line, data, model)
	}

	if len(records) == 0 {
		return nil, nil
	}

	columns := []string{}
	values := [][]interface{}{}
	for key := range records[0] {
		columns = append(columns, key)
	}
	for _, record := range records {
		value := []interface{}{}
		for _, field := range columns {
			value = append(value, record[field])
		}
		values = append(values, value)
	}
	return columns, values
}

// castByMapping 按映射表转换
func (resp *ResponseImport) castType(line int, data []map[string]string, model *Model) []map[string]interface{} {
	res := []map[string]interface{}{}
	return res
}

// castByMapping 按映射表转换
func (resp *ResponseImport) castByMapping(line int, data []map[string]string, model *Model) []map[string]interface{} {
	res := []map[string]interface{}{}
	if len(data) == 0 {
		return res
	}

	// 是否仅导入 mapping 中定义字段
	strict := false
	if strictOption, has := resp.Import.Option["strict"]; has {
		if strictOption, ok := strictOption.(bool); ok {
			strict = strictOption
		}
	}

	// 导入数据
	mappingReverse := map[string]string{}
	mapping := resp.Import.Mapping
	for _, name := range model.ColumnNames {
		if name, ok := name.(string); ok {
			if key, has := mapping[name]; has {
				mappingReverse[key] = name
			} else if !strict { // 仅导入映射表中定义字段
				if _, has := data[0][name]; has {
					mappingReverse[name] = name
				}
			}
		}
	}

	// 转换数据
	for _, record := range data {
		line++
		row := map[string]interface{}{}
		for key, name := range mappingReverse {
			row[name] = record[key]
		}

		// 数值校验（）
		errs := model.Validate(row)
		if len(errs) > 0 {
			for _, err := range errs {
				resp.Error(err.Messages[0], 400).AtLine(line).FieldIs(err.Column)
			}
			continue
		}
		res = append(res, row)
	}
	return res
}

// castByMapping 按处理器转换
func (resp *ResponseImport) castByProcess(line int, data []map[string]string, model *Model) []map[string]interface{} {
	res := []map[string]interface{}{}
	return res
}

// IsError 错误检查
func (resp *ResponseImport) IsError(line int, process interface{}) bool {
	return len(*resp.Errors) > 0
}

// Error 导入数据时错误
func (resp *ResponseImport) Error(mesage string, code int, args ...interface{}) *ErrorOfImport {
	err := &ErrorOfImport{
		Code:    code,
		Message: fmt.Sprintf(mesage, args...),
	}
	*resp.Errors = append(*resp.Errors, err)
	return err
}

// FileIs 设置出错文件
func (err *ErrorOfImport) FileIs(filename string) *ErrorOfImport {
	err.File = filename
	return err
}

// AtLine 设置出错行
func (err *ErrorOfImport) AtLine(line int) *ErrorOfImport {
	err.Line = line
	return err
}

// FieldIs 设置出错字段
func (err *ErrorOfImport) FieldIs(name string) *ErrorOfImport {
	err.Field = name
	return err
}

// ValueIs 设置出错字段
func (err *ErrorOfImport) ValueIs(value interface{}) *ErrorOfImport {
	err.Value = value
	return err
}
