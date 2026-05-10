package main

import (
	"strconv"
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

	if total < activityTableTotalWidthCM-0.01 || total > activityTableTotalWidthCM+0.01 {
		t.Fatalf("expected total width to match activity table width %.2fcm, got %.2fcm", activityTableTotalWidthCM, total)
	}
	if widths[5] >= widths[1] || widths[5] >= widths[2] || widths[5] >= widths[3] {
		t.Fatalf("expected 课时分配 column to stay narrower than main text columns, got widths %v", widths)
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

func TestRenderCoverUsesWordTemplateMeasurements(t *testing.T) {
	output := convertMarkdown(exampleMD)

	for _, want := range []string{
		"margin: (top: 2.54cm, bottom: 2.54cm, left: 2.58cm, right: 2.08cm)",
		"#v(3.20cm)",
		"#align(center)[#text(font: FONT_SONG, size: 22pt, weight: \"bold\")[教学设计方案（二）]]",
		"#v(3.25cm)",
		"columns: (auto, auto)",
		"stroke: none",
		"table.cell(align: right + bottom, stroke: (left: 0pt, right: 0pt, top: 0pt, bottom: 0pt))[#box(height: 1.50cm, inset: (bottom: 0.16cm))[#text(font: FONT_SONG, size: zh(4), weight: \"bold\")[课程名称：]]]",
		"table.cell(align: center + bottom, stroke: (left: 0pt, right: 0pt, top: 0pt, bottom: 0pt))[#box(height: 1.50cm, stroke: (bottom: 0.5pt), inset: (x: 0.45cm, bottom: 0.16cm))[#text(font: FONT_SONG, size: zh(4), weight: \"bold\")[电工基本技能训练]]]",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected reference-style cover output to contain %q", want)
		}
	}
	coverStart := strings.Index(output, "教学设计方案（二）")
	analysisStart := strings.Index(output, "#section-title[学习任务分析]")
	if coverStart < 0 || analysisStart < coverStart {
		t.Fatal("expected cover before learning-analysis section")
	}
	coverOutput := output[coverStart:analysisStart]
	if strings.Contains(coverOutput, "table.cell(stroke: (bottom: 0.5pt)") {
		t.Fatal("expected reference-style cover table cells to zero borders and draw underlines in content boxes")
	}
}

func TestLearningTaskAnalysisFieldsBlocksResourcesAndEscaping(t *testing.T) {
	output := convertMarkdown(`## 学习任务分析

学习任务: PLC 接线
课时：4
起止日期：2026 年 5 月 1 日——2026 年 5 月 2 日

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
	if strings.Contains(output, "2026 年 5 月 1 日") {
		t.Fatal("expected learning-analysis date range to omit years")
	}
	if !strings.Contains(output, "columns: (") || !strings.Contains(output, "table.cell(colspan: 6, inset: 0pt, stroke: none)[#table(") {
		t.Fatal("expected learning resources to render as a nested dynamic-width resource table")
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
	output := convertMarkdown(`---
template: "jiaoan-shicao"
---

## 教学活动设计——PLC 基本指令应用

### 认识 PLC 硬件——了解 PLC 的基本组成与接线方法

#### 活动一：PLC 硬件认知

##### 0.5H

PLC 的基本组成：CPU 模块、输入模块、输出模块、电源模块。

观察实训台上的 PLC 设备，识别各模块位置及功能。

展示 PLC 实物，讲解各模块的功能与作用。

实物展示、讲练结合
`)

	for _, want := range []string{"教学活动", "学习内容", "学生活动", "教师活动", "教学方法与手段", "课时分配"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected legacy activity output to contain %q", want)
		}
	}
	if strings.Contains(output, "教学设计方案（二）") {
		t.Fatal("did not expect template-only frontmatter to generate the new cover")
	}
}

func TestEmbeddedExampleRendersCompleteTeachingDesign(t *testing.T) {
	fm, body := parseLessonFrontMatter(exampleMD)
	sections := parseMarkdown(body)
	output := convertMarkdown(exampleMD)

	if fm.TotalHours != "8" {
		t.Fatalf("expected embedded example total_hours to be 8, got %q", fm.TotalHours)
	}
	assertSectionOrder(t, sections, []string{"学习任务分析", "教学活动设计", "学业评价"})
	if got := countActivityRows(sections); got < 8 {
		t.Fatalf("expected embedded example to contain at least 8 activity hour rows, got %d", got)
	}
	if got := sumActivityHours(sections); got != 8 {
		t.Fatalf("expected embedded example activity hours to total 8, got %.1f", got)
	}
	if got := countEvaluationRows(sections); got < 5 {
		t.Fatalf("expected embedded example to contain at least 5 evaluation rows, got %d", got)
	}
	for _, want := range []string{"教学设计方案（二）", "电工基本技能训练", "学习任务分析", "教学活动设计", "学业评价", "小结"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected embedded example output to contain %q", want)
		}
	}
}

func TestEmbeddedExampleReleaseTypstStructure(t *testing.T) {
	output := convertMarkdown(exampleMD)

	for _, want := range []string{
		"教学设计方案（二）",
		"课程名称",
		"课程属性",
		"教材名称",
		"教学班级",
		"计划总课时",
		"教师姓名",
		"使用时间",
		"学习任务分析",
		"学习任务",
		"课时",
		"起止日期",
		"一、学习任务描述",
		"二、学习目标",
		"三、学习内容",
		"四、学生情况分析",
		"五、学习资源",
		"工量具、设备",
		"耗材",
		"其它",
		"教学活动设计",
		"教学活动",
		"学习内容",
		"学生活动",
		"教师活动",
		"教学方法与手段",
		"课时分配",
		"学业评价",
		"序号",
		"考核项目",
		"考核细则",
		"考核方式",
		"小结",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected embedded example release output to contain %q", want)
		}
	}
	if strings.Contains(output, "#section-title[教学活动设计——") {
		t.Fatal("expected activity section title to omit dash and task name")
	}
	if count := strings.Count(output, "#pagebreak()"); count < 3 {
		t.Fatalf("expected at least three release pagebreaks, got %d", count)
	}
	for _, row := range []string{"[1]", "[2]", "[3]", "[4]", "[5]"} {
		if !strings.Contains(output, row) {
			t.Fatalf("expected release evaluation row marker %q", row)
		}
	}
}

func TestEvaluationProjectAndDetailsAreCentered(t *testing.T) {
	output := convertMarkdown(`## 学业评价

1. 安全文明；按规程操作；过程观察
`)

	if !strings.Contains(output, "align(center + horizon)[安全文明]") {
		t.Fatal("expected evaluation project cells to be centered")
	}
	if !strings.Contains(output, "align(center + horizon)[按规程操作]") {
		t.Fatal("expected evaluation detail cells to be centered")
	}
}

func TestEmbeddedExampleReleaseFormatSignals(t *testing.T) {
	output := convertMarkdown(exampleMD)

	for _, want := range []string{
		"flipped: false",
		"flipped: true",
		"columns: (auto, auto)",
		"box(height: 1.50cm, inset: (bottom: 0.16cm))",
		"2026 年 5 月 12 日 —— 2026 年 5 月 15 日",
		"#set page(",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected embedded example release format signal %q", want)
		}
	}
}

func TestEmbeddedExampleUsesLandscapeOnlyForActivitySection(t *testing.T) {
	output := convertMarkdown(exampleMD)

	coverIdx := strings.Index(output, "教学设计方案（二）")
	analysisIdx := strings.Index(output, "#section-title[学习任务分析]")
	activityIdx := strings.Index(output, "#section-title[教学活动设计")
	evaluationIdx := strings.Index(output, "#section-title[学业评价]")
	if !(coverIdx >= 0 && analysisIdx > coverIdx && activityIdx > analysisIdx && evaluationIdx > activityIdx) {
		t.Fatalf("expected cover < analysis < activity < evaluation, got %d %d %d %d", coverIdx, analysisIdx, activityIdx, evaluationIdx)
	}

	landscapeIdx := strings.LastIndex(output[:activityIdx], "flipped: true")
	if landscapeIdx < 0 {
		t.Fatal("expected generated Typst to switch to landscape before the activity section")
	}
	if portraitIdx := strings.LastIndex(output[:activityIdx], "flipped: false"); portraitIdx < 0 || portraitIdx > landscapeIdx {
		t.Fatal("expected portrait layout before the landscape activity switch")
	}
	if portraitAfterActivity := strings.Index(output[activityIdx:evaluationIdx], "flipped: false"); portraitAfterActivity < 0 {
		t.Fatal("expected generated Typst to switch back to portrait before evaluation")
	}
	widths := tableColumnWidthsCM([]Table{})
	total := 0.0
	for _, width := range widths {
		total += width
	}
	if total < activityTableTotalWidthCM-0.01 || total > activityTableTotalWidthCM+0.01 {
		t.Fatalf("expected activity table widths to total %.2fcm, got %.2fcm", activityTableTotalWidthCM, total)
	}
	if activityTableTotalWidthCM <= portraitTableTotalWidthCM {
		t.Fatal("expected activity tables to use a wider landscape width than portrait sections")
	}
}

func assertSectionOrder(t *testing.T, sections []DocumentSection, want []string) {
	t.Helper()
	next := 0
	for _, section := range sections {
		if next < len(want) && strings.HasPrefix(section.H2Title, want[next]) {
			next++
		}
	}
	if next != len(want) {
		t.Fatalf("expected sections to appear in order %v, got %v", want, sectionTitles(sections))
	}
}

func sectionTitles(sections []DocumentSection) []string {
	titles := make([]string, 0, len(sections))
	for _, section := range sections {
		titles = append(titles, section.H2Title)
	}
	return titles
}

func countActivityRows(sections []DocumentSection) int {
	count := 0
	for _, section := range sections {
		if !strings.HasPrefix(section.H2Title, "教学活动设计") {
			continue
		}
		for _, item := range section.Items {
			if item.Table == nil {
				continue
			}
			for _, h4 := range item.Table.H4Blocks {
				count += len(h4.H5Blocks)
			}
		}
	}
	return count
}

func sumActivityHours(sections []DocumentSection) float64 {
	total := 0.0
	for _, section := range sections {
		if !strings.HasPrefix(section.H2Title, "教学活动设计") {
			continue
		}
		for _, item := range section.Items {
			if item.Table == nil {
				continue
			}
			for _, h4 := range item.Table.H4Blocks {
				for _, h5 := range h4.H5Blocks {
					value := strings.TrimSpace(h5.Title)
					value = strings.TrimSuffix(strings.TrimSuffix(value, "H"), "h")
					hours, err := strconv.ParseFloat(value, 64)
					if err == nil {
						total += hours
					}
				}
			}
		}
	}
	return total
}

func countEvaluationRows(sections []DocumentSection) int {
	count := 0
	for _, section := range sections {
		if !strings.HasPrefix(section.H2Title, "学业评价") {
			continue
		}
		for _, line := range section.RawLines {
			line = strings.TrimSpace(line)
			if len(line) >= 3 && line[1] == '.' && line[0] >= '1' && line[0] <= '9' {
				count++
			}
		}
	}
	return count
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
