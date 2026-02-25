# 安全审查报告

**日期**: 2026-02-25
**审查范围**: presto-official-templates 全部源码、构建配置、CI 流水线

---

## 通过项

### 1. Import 白名单合规

三个 Go 文件的 import 均符合安全规范，无禁止包（`net`、`net/*`、`os/exec`、`plugin`、`debug/*`）：

- `internal/cli/cli.go` — 仅用 `encoding/json`, `flag`, `fmt`, `io`, `os`
- `gongwen/main.go` — 仅用标准库 + `goldmark` + `yaml.v3`
- `jiaoan-shicao/main.go` — 仅用 `fmt`, `strings` + 内部 `cli` 包

### 2. 无硬编码密钥/敏感信息

- 代码中无 API key、token、密码等敏感数据
- `manifest.json` 仅含公开的模板元数据

### 3. 二进制协议合规

`internal/cli/cli.go:17-49` 仅实现 `--manifest`、`--example`、`--version` 三个 flag，无多余 flag。

### 4. 第三方依赖最小化

`go.mod` 仅依赖 `goldmark` v1.7.8 和 `yaml.v3` v3.0.1，符合约束。

### 5. 构建安全

- Makefile 使用 `-trimpath -ldflags="-s -w"` 编译，剥离了调试信息和路径信息
- CI 使用 SHA256 校验和发布

### 6. 三层安全检测机制完备

- 静态分析（禁止 import 检测）
- 运行时网络沙箱（`sandbox-exec` / `unshare --net`）
- 输出格式验证（防止 HTML 注入）

---

## 发现的安全风险

### 风险 1: Typst 代码注入（中等） ✅ 已修复

`gongwen/main.go` 和 `jiaoan-shicao/main.go` 中，用户输入直接拼接进 Typst 源码，未做转义。

**修复方案**：新增 `internal/typst/escape.go`，提供 `EscapeString`（字符串上下文）和 `EscapeContent`（内容块上下文）两个转义函数，对 `\`、`"`、`#`、`]` 进行转义。已在两个模板的所有用户输入注入点应用。

### 风险 2: `io.ReadAll` 无大小限制（低） ✅ 已修复

**修复方案**：`internal/cli/cli.go` 使用 `io.LimitReader(os.Stdin, 10MB+1)` 限制输入大小，超限时输出错误并退出。

### 风险 3: `formatDate` 未转义引号（低） ✅ 已修复

与风险 1 同源，已通过 `typst.EscapeString(date)` 修复。

### 风险 4: 编译产物未加入 .gitignore（信息） ✅ 已修复

**修复方案**：创建 `.gitignore`，忽略 `presto-template-*` 和 `.DS_Store`。

### 风险 5: CI workflow 中 internal/ 变更未触发重编译（低） ✅ 已修复

**修复方案**：`.github/workflows/release.yml` 的共享文件检测正则增加 `internal/`。

---

## 总结

| 等级 | 数量 | 说明 | 状态 |
| ---- | ---- | ---- | ---- |
| 中等 | 1 | Typst 代码注入（用户输入未转义） | ✅ 已修复 |
| 低 | 3 | stdin 无大小限制、date 未转义、CI 遗漏 internal/ | ✅ 已修复 |
| 信息 | 1 | 编译产物缺 .gitignore | ✅ 已修复 |

所有风险项已于 2026-02-25 修复完毕。
