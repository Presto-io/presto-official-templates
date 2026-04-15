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
