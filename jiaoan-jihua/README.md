# 授课进度计划表模板

模板 ID：`jiaoan-jihua`

`jiaoan-jihua` 将课程元数据、学习任务、学习环节和教学内容转换为授课进度计划表 Typst 源码。它适合按周次和工作日日历生成“工学一体化课程/基本技能课程授课进度计划表”，不是实操教案正文模板。

## 最小输入

```markdown
---
school_year: "2025-2026"
semester: "第一学期"
week_range: "第1 - 2周"
major_name: "电气自动化技术"
course_name: "电气设备控制线路安装与调试"
teacher_name: "张老师"
class_name: "29WG电气3"
prepared_by: "张老师"
first_teaching_day: "2025-09-01"
daily_hours: 8
calendar_json: "presto/calendar.json"
template: "jiaoan-jihua"
---

## CA6140卧式车床电气控制线路安装与调试

### 安技教育及旧知识回顾

安技教育-1
旧知识回顾-1
```

## 输入结构

- YAML front matter 描述学年、学期、课程、教师、班级、首个授课日、每日课时和日历路径。
- `##` 表示学习任务。
- `###` 表示学习环节。
- `教学内容-数字` 表示一行教学内容及其学时；未写数字时默认 2 学时。
- `calendar_json` 默认读取 `presto/calendar.json`，用于标记调休、周末补课等实际工作日。

## 本地验证

```bash
go run . --example > /tmp/jiaoan-jihua.md
go run . < /tmp/jiaoan-jihua.md > /tmp/jiaoan-jihua.typ
typst compile /tmp/jiaoan-jihua.typ /tmp/jiaoan-jihua.pdf
```

## 常见错误

- 缺少 `##` 学习任务时，模板会输出占位提示。
- 缺少 `###` 学习环节时，模板会输出占位提示。
- `calendar_json` 指向不存在或无效 JSON 时，模板会回退到 `first_teaching_day` 连续排课并在输出中给出警告。
- 需要实操教案正文、教学活动设计或项目化教学单元时，请使用 `jiaoan-shicao`。
