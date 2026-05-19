package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestExampleOutputGoldenHash(t *testing.T) {
	fm, body := parseFrontMatter(exampleMD)
	output := convert(fm, body)
	got := fmt.Sprintf("%x", sha256.Sum256([]byte(output)))
	const want = "80c4c7d4bb923e5935e115332f86d1228b3e4688f20856dc745b9841c7bea751"
	if got != want {
		t.Fatalf("example output changed: got %s, want %s", got, want)
	}
}

func TestSingleImagesUseFloatingPlacement(t *testing.T) {
	output := convertBody(`![流程图](assets/process.png)
`)

	if !strings.Contains(output, "placement: auto") {
		t.Fatal("expected single images to use Typst floating placement")
	}
	if !strings.Contains(output, `image("assets/process.png"`) {
		t.Fatal("expected image path to be rendered")
	}
}

func TestShortTablesStayInFlowCenteredAndUnsplit(t *testing.T) {
	output := convertBody(`| 阶段 | 时间 | 负责部门 |
|------|------|----------|
| 自查自纠 | 3月1日-15日 | 各部门 |
| 集中检查 | 3月16日-31日 | 安全管理处 |

: 检查工作进度安排
`)

	if strings.Contains(output, "#place(auto, float: true)[") {
		t.Fatal("expected short tables to stay in document flow")
	}
	if !strings.Contains(output, "#block(breakable: false, width: 100%)[") {
		t.Fatal("expected short tables to stay unsplit in a full-width in-flow block")
	}
	if !strings.Contains(output, "table.cell(colspan: 3, align: center, stroke: none, inset: 0pt)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)检查工作进度安排]]]],") {
		t.Fatal("expected short table captions to use the shared table-row caption spacing")
	}
	if strings.Contains(output, "table.header(repeat: true,") {
		t.Fatal("expected short non-paginated tables to use a plain bold header row")
	}
	if !strings.Contains(output, "#align(center)[\n#table(") {
		t.Fatal("expected table to stay centered")
	}
	if !strings.Contains(output, "表1#h(1em)检查工作进度安排") {
		t.Fatal("expected table caption to stay with the table")
	}
	if strings.Contains(output, "（续）") {
		t.Fatal("expected short tables not to generate continuation captions")
	}
	if count := strings.Count(output, "#table("); count != 1 {
		t.Fatalf("expected short table not to split, got %d table chunks", count)
	}
}

func TestSignatureFlushesFloatsAndSticksToLastParagraph(t *testing.T) {
	output := convert(frontMatter{
		Title:     "测试通知",
		Author:    "测试处",
		Date:      "2026-05-06",
		Signature: true,
	}, `## 工作安排

| 阶段 | 时间 |
|------|------|
| 自查 | 3月 |
| 检查 | 4月 |

特此通知。
`)

	tableIdx := strings.Index(output, "#align(center)[\n#table(")
	flushIdx := strings.Index(output, "#place.flush()")
	stickyIdx := strings.Index(output, "#block(sticky: true, width: 100%)[")
	signatureIdx := strings.Index(output, "#block(width: 100%)[\n#align(right)[")

	if tableIdx < 0 {
		t.Fatal("expected short table to render")
	}
	if !(tableIdx < flushIdx && flushIdx < stickyIdx && stickyIdx < signatureIdx) {
		t.Fatalf("expected pending floats to flush before sticky signature, got table=%d flush=%d sticky=%d signature=%d", tableIdx, flushIdx, stickyIdx, signatureIdx)
	}
	if !strings.Contains(output[stickyIdx:signatureIdx], "特此通知。") {
		t.Fatal("expected the final paragraph to stick with the signature")
	}
	if !strings.Contains(output, "#box[\n  #set align(center)") {
		t.Fatal("expected signature content to keep centered author/date inside the right-aligned box")
	}
}

func TestLongTablesRepeatContinuationCaptionInHeader(t *testing.T) {
	var input strings.Builder
	input.WriteString("| 序号 | 内容 |\n")
	input.WriteString("|------|------|\n")
	for i := 0; i < 28; i++ {
		input.WriteString("| ")
		input.WriteString("1")
		input.WriteString(" | 事项 |\n")
	}
	input.WriteString("\n: 事项清单\n")

	output := convertBody(input.String())

	if !strings.Contains(output, "table-start-page = here().page()") {
		t.Fatal("expected long tables to track their starting page")
	}
	if !strings.Contains(output, "columns: (auto, auto),") {
		t.Fatal("expected long tables to keep auto-sized columns")
	}
	if strings.Contains(output, "1fr") {
		t.Fatal("expected long tables not to stretch the last column")
	}
	if !strings.Contains(output, "let table-caption-width = calc.max(") {
		t.Fatal("expected long tables to measure dynamic continuation caption width")
	}
	if !strings.Contains(output, "measure(text(font: FONT_FS, size: zh(3))[表1#h(1em)事项清单（续）]).width") {
		t.Fatal("expected continuation caption width to be measured from the actual caption")
	}
	if !strings.Contains(output, "(((297mm - 37mm - 35mm) / 22) - zh(3))") {
		t.Fatal("expected table caption spacing to be derived from the 22-line page grid")
	}
	if !strings.Contains(output, "table.cell(colspan: 2, align: center, stroke: none, inset: 0pt)[#context if here().page() == table-start-page") {
		t.Fatal("expected repeated table captions to use the same zero-inset table caption row")
	}
	if !strings.Contains(output, "box(width: table-caption-width)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)事项清单（续）]]]]") {
		t.Fatal("expected continuation caption to use the same centered text style and grid-derived spacing as standalone captions")
	}
	if strings.Contains(output, "pad(bottom: 8pt)") {
		t.Fatal("expected continuation captions not to use a hand-tuned fixed gap")
	}
	if strings.Contains(output, "box(width: 12em") || strings.Contains(output, "box(width: 10em") {
		t.Fatal("expected continuation captions not to use fixed em widths")
	}
	if !strings.Contains(output, "表1#h(1em)事项清单") {
		t.Fatal("expected first page table header to keep the original caption")
	}
	if !strings.Contains(output, "表1#h(1em)事项清单（续）") {
		t.Fatal("expected repeated table headers to use continuation captions")
	}
	if strings.Contains(output, "续表1") {
		t.Fatal("expected continuation captions to use 表N 表题（续）, not 续表N")
	}
	if !strings.Contains(output, "table.header(repeat: true,") {
		t.Fatal("expected long tables to repeat their header")
	}
	if count := strings.Count(output, "#table("); count != 1 {
		t.Fatalf("expected long table to remain one naturally breaking table, got %d table chunks", count)
	}
	if strings.Contains(output, "breakable: false") {
		t.Fatal("expected long tables to use normal table flow")
	}
	if strings.Contains(output, "#pagebreak(weak: true)") {
		t.Fatal("expected long tables not to force page breaks")
	}
	if strings.Contains(output, "#place(auto, float: true)[") {
		t.Fatal("expected over-one-page tables not to float")
	}
}

func TestShortTableCaptionSpacingUsesPageGrid(t *testing.T) {
	output := convertBody(`| 阶段 | 时间 |
|------|------|
| 自查 | 3月 |

: 工作安排
`)

	if !strings.Contains(output, "table.cell(colspan: 2, align: center, stroke: none, inset: 0pt)[#align(center)[#pad(bottom: (((297mm - 37mm - 35mm) / 22) - zh(3)))[#text(font: FONT_FS, size: zh(3))[表1#h(1em)工作安排]]]],") {
		t.Fatal("expected standalone table captions to use the shared table-row spacing")
	}
}
