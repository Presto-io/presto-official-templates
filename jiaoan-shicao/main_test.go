package main

import (
	"strings"
	"testing"
)

func TestPreambleEvaluatesSectionHeadingStyle(t *testing.T) {
	if !strings.Contains(preamble, "#let section-title(body)") || !strings.Contains(preamble, "#align(center)[#body]") {
		t.Fatal("expected preamble to define a centered custom section-title helper")
	}
}

func TestGenerateTypstStartsEachSectionOnNewPage(t *testing.T) {
	output := generateTypst(parseMarkdown(`## 第一部分

### 模块一——测试

#### 活动一

##### 1H

学习内容

学生活动

教师活动

教学方法

## 第二部分

### 模块二——测试

#### 活动二

##### 1H

学习内容

学生活动

教师活动

教学方法
`))

	if count := strings.Count(output, "#pagebreak()"); count < 1 {
		t.Fatalf("expected at least one automatic pagebreak between sections, got %d", count)
	}

	firstHeading := strings.Index(output, "#section-title[第一部分]")
	secondPageBreak := strings.Index(output, "#pagebreak()")
	secondHeading := strings.Index(output, "#section-title[第二部分]")
	if !(firstHeading >= 0 && secondPageBreak > firstHeading && secondHeading > secondPageBreak) {
		t.Fatal("expected second section to start after an automatic pagebreak")
	}
}

func TestTableColumnWidthsPreserveHeaderRow(t *testing.T) {
	widths := tableColumnWidthsCM([]Table{{}})

	minimums := []float64{
		headerMinWidthCM(maxInt(displayWidth("教学活动"), displayWidth("学习环节")), 0.10),
		headerMinWidthCM(displayWidth("学习内容"), 0.10),
		headerMinWidthCM(displayWidth("学生活动"), 0.10),
		headerMinWidthCM(displayWidth("教师活动"), 0.10),
		headerMinWidthCM(displayWidth("教学方法与手段"), 0.14),
		headerMinWidthCM(displayWidth("课时分配"), 0.10),
	}
	if len(widths) != len(minimums) {
		t.Fatalf("expected %d widths, got %d", len(minimums), len(widths))
	}

	total := 0.0
	for i, width := range widths {
		if width < minimums[i] {
			t.Fatalf("expected column %d to be at least %.2fcm, got %.2fcm", i, minimums[i], width)
		}
		total += width
	}

	if total < tableTotalWidthCM-0.01 || total > tableTotalWidthCM+0.01 {
		t.Fatalf("expected total width to match table width %.2fcm, got %.2fcm", tableTotalWidthCM, total)
	}
	if widths[5] >= widths[1] || widths[5] >= widths[2] || widths[5] >= widths[3] {
		t.Fatalf("expected 课时分配 column to stay narrower than main text columns, got widths %v", widths)
	}
	if widths[4] >= widths[1] || widths[4] >= widths[2] || widths[4] >= widths[3] {
		t.Fatalf("expected 教学方法与手段 column to stay narrower than main text columns, got widths %v", widths)
	}
}

func TestGenerateTypstCentersTeachingMethodAndRemovesTableGap(t *testing.T) {
	output := generateTypst(parseMarkdown(`## 第一部分

### 模块一——第一环节

#### 活动一

##### 1H

内容一

活动一

教师一

方法一

### 模块二——第二环节

#### 活动二

##### 0.5H

内容二

活动二

教师二

方法二
`))

	if !strings.Contains(output, "#block(above: 0pt, below: 0pt)[") {
		t.Fatal("expected tables to be wrapped in zero-spacing blocks")
	}
	if !strings.Contains(output, "#section-title[第一部分]\n#v(10pt)\n#block(above: 0pt, below: 0pt)[") {
		t.Fatal("expected section title to be followed by an explicit fixed gap before the table block")
	}
	if !strings.Contains(output, "#align(center)[") {
		t.Fatal("expected table to stay centered after compacting widths")
	}
	if !strings.Contains(output, "align(center + horizon)[方法一]") {
		t.Fatal("expected teaching method column to be centered")
	}
	if !strings.Contains(output, "columns: (") || !strings.Contains(output, "cm") {
		t.Fatal("expected columns to use absolute widths in cm")
	}
}

func TestGenerateTypstUsesSameColumnWidthsWithinChapter(t *testing.T) {
	output := generateTypst(parseMarkdown(`## 教学活动设计

### 安装教育——开学第一课

#### 安装教育

##### 2H

开学第一课“匠心筑梦启新程 安全护航向未来”

辨析力量一 用奋斗故事点亮理想

认真听、主动想、积极答

讲授引导法

### 讲授——GW4-40.5DW隔离开关装配与调试

#### 基本知识

##### 1H

GW4-40.5DW隔离开关结构组成及各部件功能

认真听讲，观看设备图纸

结合设备图纸和现场实物

讲授法

{pagebreak}

### 新章节——分页后重新计算

#### 第二章

##### 1H

这里是分页后的新章节内容

新的学生活动

新的教师活动

新的教学方法
`))

	firstColumnsIdx := strings.Index(output, "columns: ")
	if firstColumnsIdx < 0 {
		t.Fatal("expected first chapter to contain column specification")
	}
	firstLineEnd := strings.Index(output[firstColumnsIdx:], "\n")
	firstColumns := output[firstColumnsIdx : firstColumnsIdx+firstLineEnd]

	secondColumnsIdx := strings.Index(output[firstColumnsIdx+1:], "columns: ")
	if secondColumnsIdx < 0 {
		t.Fatal("expected second table in first chapter to contain column specification")
	}
	secondColumnsIdx += firstColumnsIdx + 1
	secondLineEnd := strings.Index(output[secondColumnsIdx:], "\n")
	secondColumns := output[secondColumnsIdx : secondColumnsIdx+secondLineEnd]

	if firstColumns != secondColumns {
		t.Fatalf("expected tables within same chapter to share identical columns, got %q and %q", firstColumns, secondColumns)
	}
}

func TestConvertMarkdownRendersYamlCoverBeforeBody(t *testing.T) {
	output := convertMarkdown(`---
course_name: 'PLC 实训 #1]'
course_attribute: 基本技能课程
textbook_name: 电气控制
class_name: 机电 1 班
total_hours: 12
teacher_name: 张三
use_time: 2026 年 5 月
---

## 学习任务分析

学习任务：PLC 接线
课时：4
起止日期：5 月 1 日
`)

	for _, label := range []string{"教学设计方案（二）", "课程名称", "课程属性", "教材名称", "教学班级", "计划总课时", "教师姓名", "使用时间"} {
		if !strings.Contains(output, label) {
			t.Fatalf("expected cover output to contain %q", label)
		}
	}
	if !strings.Contains(output, `PLC 实训 \#1\]`) {
		t.Fatal("expected cover value to be escaped")
	}
	if !strings.Contains(output, "☑基本技能课程") || !strings.Contains(output, "□工学一体化课程") {
		t.Fatal("expected 基本技能课程 selection state")
	}
}

func TestRenderCoverCourseAttributeSelectionAndMissingFields(t *testing.T) {
	basic := convertMarkdown(`---
course_name: PLC
course_attribute: 基本技能课程
---
`)
	integrated := convertMarkdown(`---
course_name: PLC
course_attribute: 工学一体化课程
---
`)
	missing := convertMarkdown(`---
course_name: 'A #set page()] \'
---
`)

	if !strings.Contains(basic, "☑基本技能课程") || !strings.Contains(basic, "□工学一体化课程") {
		t.Fatal("expected 基本技能课程 selected")
	}
	if !strings.Contains(integrated, "□基本技能课程") || !strings.Contains(integrated, "☑工学一体化课程") {
		t.Fatal("expected 工学一体化课程 selected")
	}
	for _, label := range []string{"课程名称", "课程属性", "教材名称", "教学班级", "计划总课时", "教师姓名", "使用时间"} {
		if !strings.Contains(missing, label) {
			t.Fatalf("expected missing-field cover to preserve label %q", label)
		}
	}
	for _, escaped := range []string{`\#`, `\]`, `\\`} {
		if !strings.Contains(missing, escaped) {
			t.Fatalf("expected cover output to contain escaped fragment %q", escaped)
		}
	}
}

func TestLearningTaskAnalysisFieldsBlocksResourcesAndEscaping(t *testing.T) {
	output := convertMarkdown(`## 学习任务分析

学习任务: PLC 接线
课时：4
起止日期：5 月 1 日——5 月 2 日

### 一、 学习任务描述

完成 PLC 接线。

### 二、学习目标

目标 #1]
目标二

### 三、学习内容

输入输出接线。

### 四、学生情况分析

已有电工基础。

### 五、学习资源

耗材：导线
`)

	for _, want := range []string{"PLC 接线", "5 月 1 日——5 月 2 日", "一、学习任务描述", "二、学习目标", "工量具、设备", "耗材", "其它", `目标 \#1\]`, "目标二"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected learning analysis output to contain %q", want)
		}
	}
}

func TestEvaluationOrderedListRowsDefaultsSummaryAndEscaping(t *testing.T) {
	output := convertMarkdown(`## 学业评价

1. 安全文明 #A] \；按安全规程完成接线 #B] \；过程观察 #C] \
2. 程序调试; 能独立排除常见故障; 实操考核
小结：整体达成 #好] \
`)

	for _, want := range []string{"安全文明 \\#A\\] \\\\", "按安全规程完成接线 \\#B\\] \\\\", "过程观察 \\#C\\] \\\\", "程序调试", "能独立排除常见故障", "实操考核", "小结", `整体达成 \#好\] \\`} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected evaluation output to contain %q", want)
		}
	}
	for _, row := range []string{"[1]", "[2]", "[3]", "[4]", "[5]"} {
		if !strings.Contains(output, row) {
			t.Fatalf("expected default evaluation row %q", row)
		}
	}

	sixRows := convertMarkdown(`## 学业评价

1. 项目一；细则一；方式一
2. 项目二；细则二；方式二
3. 项目三；细则三；方式三
4. 项目四；细则四；方式四
5. 项目五；细则五；方式五
6. 项目六；细则六；方式六
`)
	if !strings.Contains(sixRows, "[6]") {
		t.Fatal("expected sixth evaluation row to be appended")
	}

	shortRow := convertMarkdown(`## 学业评价

1. 安全文明；；过程观察
`)
	if !strings.Contains(shortRow, "安全文明") || !strings.Contains(shortRow, "过程观察") {
		t.Fatal("expected short evaluation row to preserve present fields")
	}
}

func TestLegacyActivityOnlyExampleStillRenders(t *testing.T) {
	output := convertMarkdown(exampleMD)

	for _, want := range []string{"教学活动", "学习内容", "学生活动", "教师活动", "教学方法与手段", "课时分配"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected legacy activity output to contain %q", want)
		}
	}
	if strings.Contains(output, "教学设计方案（二）") {
		t.Fatal("did not expect template-only frontmatter to generate the new cover")
	}
}

func TestFullCombinedInputOrderAndPagebreaks(t *testing.T) {
	output := convertMarkdown(`---
course_name: PLC 综合实训
course_attribute: 工学一体化课程
textbook_name: 电气控制
class_name: 机电 1 班
total_hours: 8
teacher_name: 李四
use_time: 2026 年 5 月
---

## 学习任务分析

学习任务：PLC 接线

### 一、学习任务描述

任务描述

### 二、学习目标

学习目标

### 三、学习内容

学习内容

### 四、学生情况分析

学生情况

### 五、学习资源

工量具、设备：万用表

## 教学活动设计——PLC 接线

### 认识 PLC——完成接线

#### 活动一

##### 1H

学习内容

学生活动

教师活动

讲练结合

## 学业评价

1. 安全文明；按规程操作；过程观察
`)

	coverIdx := strings.Index(output, "教学设计方案（二）")
	analysisIdx := strings.Index(output, "学习任务分析")
	activityIdx := strings.Index(output, "教学活动设计")
	evaluationIdx := strings.Index(output, "学业评价")
	if !(coverIdx >= 0 && analysisIdx > coverIdx && activityIdx > analysisIdx && evaluationIdx > activityIdx) {
		t.Fatalf("expected cover < analysis < activity < evaluation, got %d %d %d %d", coverIdx, analysisIdx, activityIdx, evaluationIdx)
	}
	if count := strings.Count(output, "#pagebreak()"); count < 3 {
		t.Fatalf("expected at least three pagebreaks in combined output, got %d", count)
	}
}
