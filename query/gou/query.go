package gou

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"strings"

	"github.com/go-errors/errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/helper"
	"github.com/yaoapp/gou/query/share"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/xun"
	"github.com/yaoapp/xun/dbal/query"
)

// Query Query share.DSL
type Query struct {
	QueryDSL
	Query        query.Query
	GetTableName GetTableName
	Bindings     []interface{}
	Selects      map[string]FieldNode
	AESKey       string
	STMT         string
}

// GetTableName 读取表格名称
type GetTableName = func(string) string

// Make 创建 Gou Query share.DSL
func Make(input []byte) *Query {

	var dsl QueryDSL
	err := jsoniter.Unmarshal(input, &dsl)
	if err != nil {
		exception.New("DSL 解析失败 %s", 500, err.Error()).Throw()
	}

	errs := dsl.Validate()
	if len(errs) > 0 {
		exception.New("%s", 400, errs[0]).Ctx(errs).Throw()
	}

	return &Query{QueryDSL: dsl}
}

// Read 创建 Gou Query share.DSL (输入接口)
func Read(reader io.Reader) *Query {
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, reader)
	if err != nil {
		exception.New("读取数据失败 %s", 500, err.Error()).Throw()
	}
	return Make(buf.Bytes())
}

// Open 创建 Gou Query share.DSL (文件)
func Open(filename string) *Query {
	file, err := os.Open(filename)
	if err != nil {
		exception.New("读取文件失败 %s", 500, err.Error()).Throw()
	}
	defer file.Close()
	var reader io.Reader = file
	return Read(reader)
}

// With 关联查询器
func (gou *Query) With(qb query.Query, getTableName ...GetTableName) *Query {
	gou.Query = qb.New()
	if len(getTableName) > 0 {
		return gou.TableName(getTableName[0])
	}
	return gou
}

// SetAESKey 设定 AES KEY
func (gou *Query) SetAESKey(key string) *Query {
	gou.AESKey = key
	return gou
}

// Clone 克隆对象
func (gou *Query) Clone() *Query {
	var new Query = Query{}
	new.GetTableName = gou.GetTableName
	new.AESKey = gou.AESKey
	return &new
}

// New 克隆对象
func New() *Query {
	var new Query = Query{}
	return &new
}

// TableName 绑定数据模型数据表读取方式
func (gou *Query) TableName(getTableName GetTableName) *Query {
	gou.GetTableName = getTableName
	return gou
}

// ToSQL 返回查询语句
func (gou Query) ToSQL() string {
	if gou.Query == nil {
		exception.New("未绑定数据连接", 500).Throw()
	}
	return gou.Query.ToSQL()
}

// GetBindings 返回SQL绑定数据
func (gou Query) GetBindings() []interface{} {
	if gou.Query == nil {
		exception.New("未绑定数据连接", 500).Throw()
	}
	return gou.Query.GetBindings()
}

// ==================================================
// share.DSL Interface
// ==================================================

// Load 加载查询条件
func (gou *Query) Load(data interface{}) (share.DSL, error) {

	input, err := jsoniter.Marshal(data)
	if err != nil {
		return nil, errors.Errorf("加载失败 %s", err.Error())
	}

	query := Make(input)
	query.Query = gou.Query.New()
	query.AESKey = gou.AESKey
	query.GetTableName = gou.GetTableName

	errs := query.Validate()
	if len(errs) > 0 {
		return nil, errors.Errorf("查询条件错误 %#v", errs)
	}

	query.Build()
	query.STMT = query.ToSQL()
	query.Bindings = query.GetBindings()
	query.Selects = query.mapOfSelect()
	return query, nil
}

// Run 执行查询根据查询条件返回结果
func (gou Query) Run(data maps.Map) interface{} {

	if gou.Page != nil || gou.PageSize != nil {
		return gou.Paginate(data)
	} else if gou.QueryDSL.First != nil {
		return gou.First(data)
	} else if gou.SQL == nil {
		return gou.Get(data)
	}

	sql, bindings := gou.prepare(data)
	qb := gou.Query.New()
	qb.SQL(sql, bindings...)

	// Debug模式 打印查询信息
	if gou.Debug {
		fmt.Println(sql)
		utils.Dump(bindings)
	}

	res, err := qb.DB().Exec(sql, bindings...)
	if err != nil {
		exception.New("数据查询错误 %s", 500, err.Error()).Throw()
	}
	return res
}

// GetPage get Page
func (gou Query) GetPage(data maps.Map) int {
	if gou.Page == nil {
		return 1
	}
	switch gou.Page.(type) {
	case float64, float32, int, int64, int32:
		return any.Of(gou.Page).CInt()
	case string:
		page := helper.Bind(gou.Page, data)
		return any.Of(page).CInt()
	}
	return 1
}

// GetPageSize get Page Size
func (gou Query) GetPageSize(data maps.Map) int {
	if gou.PageSize == nil {
		return 20
	}
	switch gou.PageSize.(type) {
	case float64, float32, int, int64, int32:
		return any.Of(gou.PageSize).CInt()

	case string:
		pagesize := helper.Bind(gou.PageSize, data)
		return any.Of(pagesize).CInt()
	}
	return 20
}

// SetOffset set Offset
func (gou Query) SetOffset(qb query.Query, data maps.Map) {
	if gou.Offset == nil {
		return
	}
	switch gou.Offset.(type) {
	case float64, float32, int, int64, int32:
		qb.Offset(any.Of(gou.Offset).CInt())
		break
	case string:
		offset := helper.Bind(gou.Offset, data)
		qb.Offset(any.Of(offset).CInt())
		break
	}
}

// SetLimit set Limit
func (gou Query) SetLimit(qb query.Query, data maps.Map) {
	if gou.Limit == nil && gou.SQL == nil {
		qb.Limit(100)
		return
	}
	switch gou.Limit.(type) {
	case float64, float32, int, int64, int32:
		qb.Limit(any.Of(gou.Limit).CInt())
		return
	case string:
		limit := helper.Bind(gou.Limit, data)
		qb.Limit(any.Of(limit).CInt())
		return
	}

	if gou.SQL == nil {
		qb.Limit(100)
	}
}

// Get 执行查询并返回数据记录集合
func (gou Query) Get(data maps.Map) []share.Record {

	res := []share.Record{}
	sql, bindings := gou.prepare(data)
	qb := gou.Query.New()
	gou.SetOffset(qb, data)
	gou.SetLimit(qb, data)

	qb.SQL(sql, bindings...)

	// Debug模式 打印查询信息
	if gou.Debug {
		fmt.Println(sql)
		utils.Dump(bindings)
	}

	rows := qb.MustGet()

	// 处理数据
	for _, row := range rows {
		res = append(res, gou.format(row))
	}

	return res
}

// Paginate 执行查询并返回带分页信息的数据记录数组
func (gou Query) Paginate(data maps.Map) share.Paginate {
	res := share.Paginate{}
	sql, bindings := gou.prepare(data)
	page := gou.GetPage(data)
	pageSize := gou.GetPageSize(data)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	res.Page = page
	res.PageSize = pageSize
	res.Prev = page - 1
	res.Next = page + 1
	res.Items = []share.Record{}

	total := gou.total(sql, bindings)
	res.Total = total
	res.PageCount = -1
	if total > 0 && pageSize > 0 {
		res.PageCount = int(math.Ceil(float64(total) / float64(pageSize)))
	}

	if res.Prev == 0 {
		res.Prev = -1
	}

	if res.PageCount > 0 && res.Next > res.PageCount {
		res.Next = -1
	}

	limit := pageSize
	offset := (page - 1) * pageSize
	qb := gou.Query.New()
	qb.Limit(limit).Offset(offset)
	qb.SQL(sql, bindings...)

	// Debug模式 打印查询信息
	if gou.Debug {
		fmt.Println(sql)
		utils.Dump(bindings)
	}

	rows := qb.MustGet()

	// 处理数据
	for _, row := range rows {
		res.Items = append(res.Items, gou.format(row))
	}

	return res
}

// total 计算分页 (应该在载入时准备)
func (gou Query) total(sql string, bindings []interface{}) int {
	matches := RegSelectSTMT.FindStringSubmatch(sql)
	total := -1
	if len(matches) > 0 {
		sql = strings.ReplaceAll(sql, matches[1], " COUNT(*) as `total` ")
		qb := gou.Query.New().SQL(sql, bindings...)
		// Debug模式 打印查询信息
		if gou.Debug {
			fmt.Println(sql)
			utils.Dump(bindings)
		}
		row := qb.MustFirst()
		total = row.GetInt("total")
	}
	return total
}

// First 执行查询并返回一条数据记录
func (gou Query) First(data maps.Map) share.Record {
	gou.Limit = 1
	records := gou.Get(data)
	if len(records) > 0 {
		return records[0]
	}
	return nil
}

// format 格式化输出
func (gou Query) format(row xun.R) share.Record {
	res := share.Record{}
	for key, col := range row {
		field, has := gou.Selects[key]
		val := col
		if has {
			if field.Field.IsObject {
				val = share.Record{}
				col, ok := col.(string)
				if ok && col != "" {
					err := jsoniter.Unmarshal([]byte(col), &val)
					if err != nil {
						exception.New("%s %s 数据解析错误 %s", 500, key, col, err.Error()).Throw()
					}
				}
			} else if field.Field.IsArray {
				val = []share.Record{}
				col, ok := col.(string)
				if ok && col != "" {
					err := jsoniter.Unmarshal([]byte(col), &val)
					if err != nil {
						exception.New("%s %s 数据解析错误 %s", 500, key, col, err.Error()).Throw()
					}
				}
			}
		}
		res[key] = val
	}
	return res
}

// prepare 与查询准备
func (gou *Query) prepare(data maps.Map) (string, []interface{}) {

	if gou.STMT == "" {
		exception.New("查询条件尚未加载", 404).Throw()
	}

	// 替换参数变量
	bindings := []interface{}{}
	for _, v := range gou.Bindings {
		bindings = append(bindings, v)
	}
	sql := gou.STMT
	for i := range bindings {
		bindings[i] = helper.Bind(bindings[i], data)
	}

	gou.DataOnly = helper.Bind(gou.DataOnly, data)
	return sql, bindings
}
