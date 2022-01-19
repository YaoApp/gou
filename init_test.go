package gou

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/yaoapp/gou/query/gou"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/xun/logger"
)

// TestAPIRoot
var TestAPIRoot = "/data/apis"
var TestFLWRoot = "/data/flows"
var TestPLGRoot = "/data/plugins"
var TestModRoot = "/data/models"
var TestScriptRoot = "/data/scripts"
var TestDriver = "mysql"
var TestDSN = "root:123456@tcp(127.0.0.1:3306)/gou?charset=utf8mb4&parseTime=True&loc=Local"
var TestAESKey = "123456"

func TestMain(m *testing.M) {
	TestAPIRoot = os.Getenv("GOU_TEST_API_ROOT")
	TestFLWRoot = os.Getenv("GOU_TEST_FLW_ROOT")
	TestModRoot = os.Getenv("GOU_TEST_MOD_ROOT")
	TestPLGRoot = os.Getenv("GOU_TEST_PLG_ROOT")
	TestScriptRoot = os.Getenv("GOU_TEST_SCRIPT_ROOT")
	TestDriver = os.Getenv("GOU_TEST_DB_DRIVER")
	TestDSN = os.Getenv("GOU_TEST_DSN")
	TestAESKey = os.Getenv("GOT_TEST_AES_KEY")

	// 数据库连接
	switch TestDriver {
	case "sqlite3":
		capsule.AddConn("primary", "sqlite3", TestDSN).SetAsGlobal()
		break
	default:
		capsule.AddConn("primary", "mysql", TestDSN).SetAsGlobal()
		break
	}
	SetModelLogger(os.Stdout, logger.LevelDebug)

	// 注册数据分析引擎
	RegisterEngine("test-db", &gou.Query{
		Query: capsule.Query(),
		GetTableName: func(s string) string {
			if mod, has := Models[s]; has {
				return mod.MetaData.Table.Name
			}
			exception.New("%s 数据模型尚未加载", 404).Throw()
			return s
		},
		AESKey: TestAESKey,
	})

	// 加载模型
	LoadModel("file://"+path.Join(TestModRoot, "user.json"), "user")
	LoadModel("file://"+path.Join(TestModRoot, "manu.json"), "manu")
	LoadModel("file://"+path.Join(TestModRoot, "address.json"), "address")
	LoadModel("file://"+path.Join(TestModRoot, "role.json"), "role")
	LoadModel("file://"+path.Join(TestModRoot, "friends.json"), "friends")
	LoadModel("file://"+path.Join(TestModRoot, "user_roles.json"), "user_roles")

	// 加载插件
	LoadPlugin(path.Join(TestPLGRoot, "user"), "user")
	defer SelectPlugin("user").Client.Kill()

	// 加载 API
	LoadAPI("file://"+path.Join(TestAPIRoot, "user.http.json"), "user")
	LoadAPI("file://"+path.Join(TestAPIRoot, "manu.http.json"), "manu")

	// 加载 Flow
	JavaScriptVM.Load(path.Join(TestScriptRoot, "test.js"), "app.test") // 加载全局脚本

	LoadFlow("file://"+path.Join(TestFLWRoot, "latest.flow.json"), "latest").
		LoadScript("file://"+path.Join(TestFLWRoot, "latest.rank.js"), "rank").
		LoadScript("file://"+path.Join(TestFLWRoot, "latest.count.js"), "count")

	LoadFlow("file://"+path.Join(TestFLWRoot, "stat.flow.json"), "stat")

	LoadFlow("file://"+path.Join(TestFLWRoot, "script.flow.json"), "script").
		LoadScript("file://"+path.Join(TestFLWRoot, "script.rank.js"), "rank").
		LoadScript("file://"+path.Join(TestFLWRoot, "script.sort.js"), "sort")

	LoadFlow("file://"+path.Join(TestFLWRoot, "arrayset.flow.json"), "arrayset").
		LoadScript("file://"+path.Join(TestFLWRoot, "arrayset.array.js"), "array")

	LoadFlow("file://"+path.Join(TestFLWRoot, "user", "info.flow.json"), "user.info").
		LoadScript("file://"+path.Join(TestFLWRoot, "user", "info.data.js"), "data")

	// 加密密钥
	LoadCrypt(fmt.Sprintf(`{"key":"%s"}`, TestAESKey), "AES")
	LoadCrypt(`{}`, "PASSWORD")

	// Run test suites
	exitVal := m.Run()

	// 释放资源
	KillPlugins()

	// we can do clean up code here
	os.Exit(exitVal)

}
