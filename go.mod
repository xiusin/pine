module github.com/xiusin/pine

go 1.13

require (
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/CloudyKit/jet v2.1.3-0.20180809161101-62edd43e4f88+incompatible
	github.com/aliyun/aliyun-oss-go-sdk v2.0.4+incompatible
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/betacraft/yaag v1.0.0
	github.com/casbin/casbin v1.9.1
	github.com/dgraph-io/badger v1.6.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/fatih/color v1.7.0
	github.com/flosch/pongo2 v0.0.0-20190707114632-bbf5a6c351f4
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/gookit/color v1.2.2
	github.com/gorilla/schema v1.1.0
	github.com/satori/go.uuid v1.2.0
	github.com/smartystreets/assertions v0.0.0-20180927180507-b2de0cb4f26d
	github.com/smartystreets/goconvey v1.6.4
	github.com/xiusin/logger v0.0.0-00010101000000-000000000000
	go.etcd.io/bbolt v1.3.3
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
)

replace github.com/xiusin/logger => ../logger
