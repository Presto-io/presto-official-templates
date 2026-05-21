# Presto 模板开发指南

本文件为 stub。完整规范请参阅中心文档：

https://github.com/Presto-io/Presto-Homepage/blob/main/docs/CONVENTIONS.md

## 模板协议兜底要求

- 默认模式：stdin Markdown -> stdout Typst。
- stdout 中的 Typst 必须非空；空字符串或纯空白输出视为转换失败。
- 转换失败必须向 stderr 写入错误信息，并以非 0 状态退出。
- 模板不得吞掉异常后返回空 Typst，因为 Presto 会把退出码为 0 的 stdout 交给 Typst 编译。
- `--manifest`、`--example`、`--version`、`--info` 不受默认转换模式校验影响。
