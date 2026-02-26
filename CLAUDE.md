# Presto Official Templates

> 组织级规则见 `../Presto-homepage/docs/ai-guide.md`
> 模板开发规范见 `../Presto-homepage/docs/CONVENTIONS.md`

## 仓库结构

这是一个 monorepo，包含多个官方模板。每个模板是独立的 `main` package，在自己的子目录下。

```
presto-official-templates/
├── gongwen/              # 类公文模板
│   ├── main.go
│   ├── template_head.typ
│   ├── example.md
│   └── manifest.json
├── jiaoan-shicao/        # 实操教案模板
│   ├── main.go
│   ├── example.md
│   └── manifest.json
├── go.mod                # 共享 Go module
├── go.sum
└── Makefile
```

## 关键约束

- 不要修改模板二进制协议（stdin/stdout 接口）
- 不要引入新的第三方 Go 依赖（只用 goldmark + yaml.v3 + 标准库）
- 禁止 import `net`、`net/*`、`os/exec`、`plugin`、`debug/*`（安全规范）
- 每个模板是独立的 main package，在自己的子目录下
- `go.mod` 放在仓库根目录，构建时需要 `cd` 进子目录

## 二进制协议

每个模板编译后的二进制必须且只能支持以下四种调用：

| 调用方式 | 行为 |
|---------|------|
| `cat input.md \| ./binary` | stdin 读 Markdown，stdout 输出 Typst 源码 |
| `./binary --manifest` | stdout 输出 manifest.json |
| `./binary --example` | stdout 输出 example.md |
| `./binary --version` | stdout 输出版本号（从 manifest.json 解析） |

不得添加其他 flag（如 `-o`、`-p`、`-v`、`-h` 等）。

## 构建与测试

```bash
# 构建所有模板
make build-all

# 测试（含安全检测 + 功能测试）
make test

# 仅安全检测
make test-security

# 安装到 Presto 预览
make preview NAME=gongwen
```

## 添加新模板

1. 创建子目录 `<name>/`
2. 实现 `main.go`（遵循二进制协议，只用允许的 import）
3. 创建 `manifest.json` 和 `example.md`（编译时嵌入）
4. 在 `Makefile` 的 `TEMPLATES` 变量中添加模板名
5. 在 `.github/workflows/release.yml` 的 matrix 中添加模板名
6. 运行 `make test` 验证（安全 + 功能）
