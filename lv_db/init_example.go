// Package lv_db 数据库引擎包
// 
// 这个示例展示了如何正确使用优化后的方言注册机制
// 以实现业务模块只导入必要的数据库驱动
package lv_db

// 示例：如何在业务模块中注册SQLite方言
// 
// 业务模块只需要导入：
// 1. lv_db包（核心引擎）
// 2. sqlite_dialector包（需要的方言）
// 3. 对应的GORM驱动包（gorm.io/driver/sqlite）
//
// 不需要导入其他数据库驱动（如mysql、postgres），从而减少不必要的依赖
//
// 示例代码：
//
// import (
//     "github.com/lostvip-com/lv_framework/lv_db"
//     _ "github.com/lostvip-com/lv_framework/lv_db/lv_dialector/sqlite_dialector"
//     _ "gorm.io/driver/sqlite" // 只导入需要的驱动
// )
//
// func init() {
//     // 注册SQLite方言
//     lv_db.GetInstance().RegisterDialector("sqlite", func() lv_dialector.Dialector {
//         return &lv_dialector.SQLiteDialector{}
//     })
//     
//     // 或者注册为默认方言
//     // lv_db.GetInstance().RegisterDefaultDialector("sqlite", func() lv_dialector.Dialector {
//     //     return &lv_dialector.SQLiteDialector{}
//     // })
// }
//
// func main() {
//     // 使用已注册的方言创建数据库连接
//     // 配置文件中定义的数据源会自动使用注册的方言
//     db := lv_db.GetInstance().GetDefault()
//     
//     // 执行数据库操作
//     // ...
// }
//
// 注意：
// 1. 每个方言只需要注册一次，多个数据源可以共享同一个方言
// 2. 未注册的方言在使用时会报错，避免了隐式依赖
// 3. 业务模块可以根据需要注册不同的方言组合