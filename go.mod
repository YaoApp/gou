module github.com/yaoapp/gou

go 1.23.0

toolchain go1.23.4

require (
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/fatih/color v1.16.0
	github.com/fsnotify/fsnotify v1.6.0
	github.com/gabriel-vasile/mimetype v1.4.4
	github.com/gin-gonic/gin v1.10.0
	github.com/go-errors/errors v1.4.2
	github.com/go-redis/redis/v8 v8.11.5
	github.com/gorilla/websocket v1.5.0
	github.com/hashicorp/go-hclog v1.5.0
	github.com/hashicorp/go-plugin v1.6.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/json-iterator/go v1.1.12
	github.com/miekg/dns v1.1.48
	github.com/robfig/cron/v3 v3.0.0
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/buntdb v1.3.0
	github.com/yaoapp/kun v0.9.0
	github.com/yaoapp/xun v0.0.0-00010101000000-000000000000
	golang.org/x/crypto v0.36.0
	gopkg.in/yaml.v3 v3.0.1
	rogchap.com/v8go v0.8.0
)

require (
	github.com/TylerBrock/colorjson v0.0.0-20200706003622-8a50f05110d2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.22.0 // indirect
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/jmoiron/sqlx v1.3.1 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lib/pq v1.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.6 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/tidwall/btree v1.4.2 // indirect
	github.com/tidwall/gjson v1.14.3 // indirect
	github.com/tidwall/grect v0.1.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tidwall/rtred v0.1.2 // indirect
	github.com/tidwall/tinyqueue v0.1.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.mongodb.org/mongo-driver v1.13.0
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	google.golang.org/grpc v1.69.2
	google.golang.org/protobuf v1.36.1 // indirect
)

require (
	github.com/evanw/esbuild v0.19.3
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible
	github.com/google/uuid v1.6.0
	github.com/qdrant/go-client v1.12.0
	github.com/sergi/go-diff v1.3.1
	github.com/watchfultele/jsonrepair v0.0.0-20250207052432-e4397ed42611
	golang.org/x/image v0.18.0
)

require (
	github.com/bytedance/sonic v1.11.9 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.org/x/arch v0.8.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241230172942-26aa7a208def // indirect
)

replace github.com/yaoapp/kun => ../kun

replace github.com/yaoapp/xun => ../xun

replace rogchap.com/v8go => ../v8go
