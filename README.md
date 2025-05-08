# JSON Template with Go

一个 Go 的 JSON 模板引擎，可以将 Go 的表达式嵌入到 JSON 中。它支持条件语句、循环、函数调用等功能，可以用于生成动态的 JSON 数据。

## 执行模型

`模板+数据` -> `编译` -> `解析执行` -> `JSON`