package gou

// Process 运行器
type Process struct {
	Name    string
	Type    string
	Class   string
	Method  string
	Args    []interface{}
	Handler ProcessHandler
}

// ProcessHandler 处理程序
type ProcessHandler func(process *Process) interface{}

// ThirdHandlers 第三方处理器
var ThirdHandlers = map[string]ProcessHandler{}

// ModelHandlers 模型运行器
var ModelHandlers = map[string]ProcessHandler{
	"find":         processFind,
	"get":          processGet,
	"paginate":     processPaginate,
	"create":       processCreate,
	"update":       processUpdate,
	"save":         processSave,
	"delete":       processDelete,
	"destroy":      processDestroy,
	"insert":       processInsert,
	"updatewhere":  processUpdateWhere,
	"deletewhere":  processDeleteWhere,
	"destroywhere": processDestroyWhere,
}
