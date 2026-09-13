// Package lv_db 数据库引擎包
//
// 本示例说明:应用/业务方如何在不改动框架的前提下,自定义或扩展数据库方言。
// 核心包 lv_dialector 只持注册表(零驱动依赖),具体方言一律经注册进入。
package lv_db

// ─────────────────────────────────────────────────────────────────
// 方式一(推荐):自定义方言子包,init() 自注册,blank-import 引入
//
// 适合:接入框架未内置的数据库,或为兼容库(如 TiDB/OceanBase)定义
// 独立参数集。与框架仓库内 lv_db/dialect/<driver> 子包的做法完全一致。
//
// 	// myapp/dialect/mydb/dialector.go —— 用户自己的包
// 	package mydb
//
// 	import (
// 	    "gorm.io/gorm"
//
// 	    "github.com/lostvip-com/lv_framework/lv_db/lv_dialector"
// 	)
//
// 	func init() {
// 	    lv_dialector.RegisterDialector("mydb", func() lv_dialector.Dialector {
// 	        return &Dialector{}
// 	    })
// 	    // 可选:登记默认连接参数(会以 ?k=v 拼入 URL,自定义参数可覆盖;
// 	    // DSN 不支持查询串的数据库留空即可,URL 将原样透传)
// 	    lv_dialector.DefaultParamsProvider["mydb"] = &ParamsProvider{}
// 	}
//
// 	type Dialector struct{}
//
// 	func (d *Dialector) GetName() string { return "mydb" }
//
// 	func (d *Dialector) NewDialector(url string) gorm.Dialector {
// 	    return gormDialector{dsn: url} // 实现 gorm.Dialector,参考 gorm 官方
// 	    // "write driver" 文档;若目标库已有 database/sql 驱动,约 40 行即可
// 	    // (可参考框架内 lv_db/dialect/tdengine 子包的完整写法)
// 	}
//
// 	// main.go(或 bootstrap 包)blank-import 引入,init 自动注册:
// 	// import _ "myapp/dialect/mydb"
//
// 	// application-dev.yml 即可直接使用:
// 	//   datasource:
// 	//     db-custom:
// 	//       driver: mydb
// 	//       url: ...
// ─────────────────────────────────────────────────────────────────
// 方式二:使用框架内置方言(按需 blank-import,不引入的不参与链接)
//
// 	import (
// 	    _ "github.com/lostvip-com/lv_framework/lv_db/dialect/mysql"
// 	    _ "github.com/lostvip-com/lv_framework/lv_db/dialect/tdengine"
// 	)
//
// 未引入的方言(如 sqlite/postgres)不注册、不链接 —— 未注册的 driver
// 在建连时会返回明确错误,避免隐式依赖。
// ─────────────────────────────────────────────────────────────────
// 方式三:绕过方言,直接注册现成的 *gorm.DB(逃生通道)
//
// 适合:连接由外部构造(已有 *sql.DB、代理、测试桩等)的场景:
//
// 	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
// 	lv_db.GetInstance().RegisterDB("db-custom", gdb)
//
// 注意事项:
// 1. 注册必须发生在 InitDataSources() 之前 —— 放在自己包的 init() 里最稳
//    (包初始化先于 main,时序天然保证);
// 2. DefaultParamsProvider 为普通 map,请在 init()(单线程)阶段写入;
// 3. 一个方言可被多个数据源共享;数据源间用 application.datasource.<name> 区分。
