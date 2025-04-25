请严格遵循以下规范进行代码编写：

## 项目简介
项目名称是 JTGO（JSON Template with Go），这是一个基于 Go 的 JSON 模板引擎，可以将 Go 的表达式嵌入到 JSON 中。它支持条件语句、循环、函数调用等功能，可以用于生成动态的 JSON 数据。

## 目录结构
```
/engines    // 模板引擎核心的实现
/ds         // 数据结构定义
/examples   // 示例模板、数据和示例代码
/util       // 工具函数
/werror     // Error 定义
```

## 命名规范
- 采用驼峰命名法（CamelCase）
- 对于全大写的专有名词，保持其大写形式，例如：`JSONTemplate`、`FastHTTPClient`

## 编程要求
- 使用 Go 1.22 及以下的版本支持的特性
- 始终将代码分解为模块和组件，以便可以在整个项目中轻松重复使用。
- JSON 序列化相关操作不要使用标准库，请使用 `github.com/bytedance/sonic`
- 使用 `go:embed` 来嵌入模板文件

## 注释要求
- 注释使用中文