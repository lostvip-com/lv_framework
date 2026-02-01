# lv_framework

<p align="center">
  <h3 align="center">Copyright © 2019 <a href="https://lostvip.com">lostvip.com</a></h3>
  <p align="center">
    <img alt="License" src="https://img.shields.io/badge/license-Apache%202.0-blue.svg">
  </p>
</p>

---

## 开源许可

**Copyright 2019 lostvip.com**

本项目采用 **Apache License 2.0** 开源许可证。

### 版权声明

```
Copyright 2019 lostvip.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

完整的许可证文本请参阅项目根目录的 [LICENSE](../../LICENSE) 文件。

---

## 框架简介

**lv_framework** 是一个 Go 语言轻量级快速开发框架，摒弃过度封装，代码风格极尽简洁，适合中小项目使用。

### 核心特性

- ✅ **Spring Boot 风格** - 项目结构模仿 Spring Boot，对 Java 开发人员友好
- ✅ **MyBatis 风格 SQL** - 支持 SQL 与 Go 代码分离（基于 GORM + template 语法）
- ✅ **环境变量支持** - YAML 配置文件支持 `${VAR:default}` 表达式
- ✅ **多数据库支持** - MySQL、SQLite3、PostgreSQL，可自行扩展
- ✅ **热加载模板** - Gin 模板引擎支持缓存 TTL 配置

---

## 目录结构

```
lv_framework/
├── lv_cache/          # 通用缓存（Redis/RAM）
├── lv_conf/           # 通用配置管理
├── lv_db/             # 数据库相关
│   ├── lv_batis/      # MyBatis 风格 SQL 查询
│   ├── lv_dao/        # 泛型 CRUD
│   └── lv_dialector/  # 数据库方言
├── lv_global/         # 全局常量
├── lv_log/            # 统一日志接口
├── utils/             # 工具类集合
│   ├── lv_arr/        # 数组工具
│   ├── lv_conv/       # 类型转换
│   ├── lv_err/        # 错误处理
│   ├── lv_file/       # 文件操作
│   ├── lv_net/        # HTTP 客户端
│   ├── lv_reflect/    # 反射工具
│   ├── lv_secret/     # 加密解密
│   └── ...
└── web/               # Web 组件
    ├── lv_dto/        # 响应 DTO
    ├── middleware/    # Gin 中间件
    ├── router/        # 路由管理
    └── server/        # HTTP 服务器
```

---

## 引入方式

```bash
go get github.com/lostvip-com/lv_framework
```

---

## 商业支持

| 服务类型 | 说明 | 联系方式 |
|---------|------|---------|
| 社区版 | 免费使用，社区支持 | [GitHub Issues](https://github.com/lostvip-com/ruoyi-go/issues) |
| 专业版 | 付费企业技术支持 | [联系我们](https://github.com/lostvip-com) |
| 企业版 | 定制开发、企业培训 | [联系我们](https://github.com/lostvip-com) |

---

## 项目链接

- **GitHub**: https://github.com/lostvip-com/ruoyi-go
- **官网**: https://lostvip.com
- **微信公众号**: lostvip666
- **QQ群**: 43862272

---

## 致谢

本项目借鉴和使用了以下优秀开源项目：

- [GORM](https://gorm.io/) - ORM 框架
- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [Viper](https://github.com/spf13/viper) - 配置管理
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [dotsql](https://github.com/qustavo/dotsql) - SQL 文件解析
