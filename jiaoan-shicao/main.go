package main

import (
	_ "embed"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/Presto-io/presto-official-templates/internal/cli"
	"github.com/Presto-io/presto-official-templates/internal/typst"
)

//go:embed manifest.json
var manifestJSON string

//go:embed example.md
var exampleMD string

func main() {
	cli.Run(manifestJSON, exampleMD, func(input string) string {
		sections := parseMarkdown(input)
		return generateTypst(sections)
	})
}

const preamble = `// 中文字号转换函数
#import "@preview/pointless-size:0.1.2": zh
#import "@preview/cuti:0.2.1": show-cn-fakebold
#show: show-cn-fakebold

// 定义常用字体名称
#let FONT_XBS = ("FZXiaoBiaoSong-B05") // 方正小标宋
#let FONT_HEI = ("STHeiti") // 黑体
#let FONT_FS = ("STFangsong") // 仿宋
#let FONT_KAI = ("STKaiti") // 楷体
#let FONT_SONG = ("STSong") // 宋体

#set page(
  paper: "a4",
  flipped: true,
  margin: (top: 2.54cm, bottom: 2.54cm, left: 2.58cm, right: 2.08cm)
)

#set text(
  lang: "zh",
  font: FONT_SONG,
  size: zh(5),
  hyphenate: false,
  tracking: -0.3pt,
  cjk-latin-spacing: auto
)

#let section-title(body) = block(above: 0pt, below: 0pt, width: 100%)[
  #set text(font: FONT_SONG, size: zh(4))
  #align(center)[#body]
]
`

const tableTotalWidthCM = 25.04
const sectionHeadingGap = "10pt"

// H5Block 存储五级标题及其内容
type H5Block struct {
	Title   string
	Content []string
}

// H4Block 存储四级标题及其下的所有五级标题块
type H4Block struct {
	Title    string
	H5Blocks []H5Block
}

// Table 存储一个三级标题定义的表格
type Table struct {
	H3Part1  string
	H3Part2  string
	H4Blocks []H4Block
}

// DocumentSection 存储一个二级标题定义的内容区域
type DocumentSection struct {
	H2Title string
	Items   []SectionItem
}

// SectionItem 表示二级标题下的一项内容，可以是表格或分页符。
type SectionItem struct {
	Table     *Table
	PageBreak *PageBreak
}

// PageBreak 表示显式分页。
type PageBreak struct {
	Weak bool
}

type tableContinuation struct {
	H3Part1 string
	H3Part2 string
	H4Title string
}

// parseMarkdown 将 markdown 字符串解析为 DocumentSection 结构体切片
func parseMarkdown(content string) []DocumentSection {
	lines := strings.Split(content, "\n")
	var sections []DocumentSection
	var currentSection *DocumentSection
	var currentTable *Table
	var currentH4 *H4Block
	var currentH5 *H5Block
	var continuation *tableContinuation

	for _, line := range lines {
		line = strings.TrimRight(line, "\r") // 兼容 Windows 换行符
		if pageBreak, ok := parsePageBreakMarker(line); ok {
			if currentSection == nil {
				continue
			}
			if currentTable != nil {
				continuation = &tableContinuation{
					H3Part1: currentTable.H3Part1,
					H3Part2: currentTable.H3Part2,
				}
				if currentH4 != nil {
					continuation.H4Title = currentH4.Title
				}
			} else {
				continuation = nil
			}
			currentSection.Items = append(currentSection.Items, SectionItem{PageBreak: &pageBreak})
			currentTable = nil
			currentH4 = nil
			currentH5 = nil
			continue
		}

		if strings.HasPrefix(line, "## ") {
			sections = append(sections, DocumentSection{H2Title: strings.TrimSpace(line[3:])})
			currentSection = &sections[len(sections)-1]
			currentTable = nil
			currentH4 = nil
			currentH5 = nil
			continuation = nil
		} else if strings.HasPrefix(line, "### ") {
			if currentSection == nil {
				continue
			}
			title := strings.TrimSpace(line[4:])

			var parts []string
			// Try splitting with various separators, from longest to shortest.
			separators := []string{"——", "—", " - ", "-"}
			found := false
			for _, sep := range separators {
				parts = strings.SplitN(title, sep, 2)
				if len(parts) == 2 {
					found = true
					break
				}
			}

			if !found {
				parts = []string{title, ""}
			}

			currentTable = &Table{H3Part1: strings.TrimSpace(parts[0]), H3Part2: strings.TrimSpace(parts[1])}
			currentSection.Items = append(currentSection.Items, SectionItem{Table: currentTable})
			currentH4 = nil
			currentH5 = nil
			continuation = nil
		} else if strings.HasPrefix(line, "#### ") {
			if currentSection == nil {
				continue
			}
			if currentTable == nil && continuation != nil {
				currentTable, currentH4 = startContinuationTable(currentSection, continuation, false)
				continuation = nil
			}
			if currentTable == nil {
				continue
			}
			title := strings.TrimSpace(line[5:])
			currentTable.H4Blocks = append(currentTable.H4Blocks, H4Block{Title: title})
			currentH4 = &currentTable.H4Blocks[len(currentTable.H4Blocks)-1]
			currentH5 = nil
		} else if strings.HasPrefix(line, "##### ") {
			if currentSection == nil {
				continue
			}
			if currentTable == nil && continuation != nil {
				currentTable, currentH4 = startContinuationTable(currentSection, continuation, true)
				continuation = nil
			}
			if currentH4 == nil {
				continue
			}
			title := strings.TrimSpace(line[6:])
			currentH4.H5Blocks = append(currentH4.H5Blocks, H5Block{Title: title})
			currentH5 = &currentH4.H5Blocks[len(currentH4.H5Blocks)-1]
			// Initialize with one empty content block, ready to be filled.
			currentH5.Content = []string{""}
		} else {
			if currentH5 != nil {
				if strings.TrimSpace(line) == "" {
					// Empty line: if the current block has content, prepare for a new block.
					lastIdx := len(currentH5.Content) - 1
					if lastIdx >= 0 && currentH5.Content[lastIdx] != "" {
						currentH5.Content = append(currentH5.Content, "")
					}
				} else {
					// Non-empty line: append to the current block.
					lastIdx := len(currentH5.Content) - 1
					if lastIdx < 0 {
						currentH5.Content = append(currentH5.Content, "")
						lastIdx = 0
					}

					if currentH5.Content[lastIdx] == "" {
						currentH5.Content[lastIdx] = line
					} else {
						// 使用真实的换行符，Typst 会在内容块中将其渲染为换行。
						currentH5.Content[lastIdx] += "\n" + line
					}
				}
			}
		}
	}

	// Clean up trailing empty content block if any
	for i := range sections {
		for j := range sections[i].Items {
			table := sections[i].Items[j].Table
			if table == nil {
				continue
			}
			for k := range table.H4Blocks {
				for l := range table.H4Blocks[k].H5Blocks {
					h5 := &table.H4Blocks[k].H5Blocks[l]
					if len(h5.Content) > 0 && h5.Content[len(h5.Content)-1] == "" {
						h5.Content = h5.Content[:len(h5.Content)-1]
					}
				}
			}
		}
	}

	return sections
}

// generateTypst 根据解析出的结构体生成 typst 格式字符串
func generateTypst(sections []DocumentSection) string {
	var sb strings.Builder
	sb.WriteString(preamble)

	for sectionIdx, section := range sections {
		if sectionIdx > 0 {
			sb.WriteString("\n#pagebreak()\n\n")
		}
		sb.WriteString(fmt.Sprintf("\n#section-title[%s]\n#v(%s)\n", typst.EscapeContent(section.H2Title), sectionHeadingGap))

		chapterColumnSpecs := sectionColumnSpecs(section)
		chapterIdx := 0

		for _, item := range section.Items {
			if item.PageBreak != nil {
				if item.PageBreak.Weak {
					sb.WriteString("#pagebreak(weak: true)\n\n")
				} else {
					sb.WriteString("#pagebreak()\n\n")
				}
				chapterIdx++
				continue
			}

			if item.Table == nil || len(item.Table.H4Blocks) == 0 {
				continue
			}

			table := item.Table
			sb.WriteString("#block(above: 0pt, below: 0pt)[\n")
			sb.WriteString("  #align(center)[\n")
			sb.WriteString("    #table(\n")
			sb.WriteString(fmt.Sprintf("      columns: %s,\n", chapterColumnSpecs[chapterIdx]))
			sb.WriteString("      stroke: 0.5pt,\n")
			sb.WriteString("      align: center + horizon,\n")

			// 表格第一行
			sb.WriteString(fmt.Sprintf("      [*学习环节*], [*%s*], [*学习单元*], table.cell(colspan: 3)[*%s*],\n", typst.EscapeContent(table.H3Part1), typst.EscapeContent(table.H3Part2)))

			// 表格第二行
			sb.WriteString("      [教学活动], [学习内容], [学生活动], [教师活动], [教学方法与手段], [课时分配],\n")

			h4Counter := 1 // Reset for each table (H3)

			// 内容行
			for _, h4 := range table.H4Blocks {
				if len(h4.H5Blocks) == 0 {
					continue
				}
				// 为当前 H4 构建单元格内容矩阵（每行 5 列：content0, content1, content2, teachingMethods, h5.Title）
				nRows := len(h4.H5Blocks)
				cols := 5
				cellContents := make([][]string, nRows)
				for i := 0; i < nRows; i++ {
					h5 := h4.H5Blocks[i]
					cellContents[i] = make([]string, cols)
					cellContents[i][0] = typst.EscapeContent(getContentLine(h5.Content, 0))
					cellContents[i][1] = typst.EscapeContent(getContentLine(h5.Content, 1))
					cellContents[i][2] = typst.EscapeContent(getContentLine(h5.Content, 2))
					cellContents[i][3] = typst.EscapeContent(getContentLine(h5.Content, 3)) // 教学方法，渲染时会替换换行
					cellContents[i][4] = typst.EscapeContent(h5.Title)
				}

				// 初始化 rowspan 矩阵，默认每个单元格 rowspan = 1
				rowspans := make([][]int, nRows)
				for i := 0; i < nRows; i++ {
					rowspans[i] = make([]int, cols)
					for j := 0; j < cols; j++ {
						rowspans[i][j] = 1
					}
				}

				// 处理包含 "同上" 的单元格：与正上方起始单元格合并（递归合并链）
				for col := 0; col < cols; col++ {
					for i := 0; i < nRows; i++ {
						if strings.Contains(strings.TrimSpace(cellContents[i][col]), "同上") {
							// 找到上方最近的起始单元格（rowspan != 0）
							k := i - 1
							for k >= 0 && rowspans[k][col] == 0 {
								k--
							}
							if k >= 0 {
								rowspans[k][col]++
								rowspans[i][col] = 0 // 标记为已被合并，输出时跳过
							} else {
								// 若没有上方可合并的单元格（首行），保留为空字符串，不合并
								cellContents[i][col] = ""
								rowspans[i][col] = 1
							}
						}
					}
				}

				numberedH4Title := fmt.Sprintf("%d.%s", h4Counter, typst.EscapeContent(h4.Title))
				h4Counter++

				// 为每列在输出时维护独立序号计数器（H4 内重置）
				counters := [3]int{1, 1, 1}

				// 输出每一行，依据 rowspans 决定是否输出或输出带 rowspan 的单元格
				for i := 0; i < nRows; i++ {
					// 第一列（H4 标题）只在第一行输出，并带有整体 rowspan
					if i == 0 {
						sb.WriteString(fmt.Sprintf("      table.cell(rowspan: %d)[%s],", nRows, numberedH4Title))
					}

					// 对应三列内容 + 教学方法 + 课时分配
					for col := 0; col < cols; col++ {
						rs := rowspans[i][col]
						if rs == 0 {
							// 被上方合并，跳过输出该单元格
							continue
						}

						content := cellContents[i][col]
						if col <= 2 {
							content, counters[col] = formatNumberedContent(content, counters[col])
						} else if col == 3 {
							// 教学方法列，替换换行为双换行
							if strings.TrimSpace(content) != "" {
								content = strings.ReplaceAll(content, "\n", "\n\n")
							}
						}

						// 仅在 rowspan > 1 时使用 table.cell
						if rs > 1 {
							var attrs []string
							attrs = append(attrs, fmt.Sprintf("rowspan: %d", rs))
							if align := cellAlign(col, content); align != "" {
								attrs = append(attrs, fmt.Sprintf("align: %s", align))
							}
							sb.WriteString(fmt.Sprintf("      table.cell(%s)[%s],", strings.Join(attrs, ", "), content))
						} else {
							// rowspan == 1 时，不使用 table.cell，对齐通过 align() 包裹
							if align := cellAlign(col, content); align != "" {
								sb.WriteString(fmt.Sprintf("      align(%s)[%s],", align, content))
							} else {
								sb.WriteString(fmt.Sprintf("      [%s],", content))
							}
						}
					}
					sb.WriteString("\n")
				}
			}
			sb.WriteString("    )\n")
			sb.WriteString("  ]\n")
			sb.WriteString("]\n")
		}
	}
	return sb.String()
}

// getContentLine 安全地获取内容行，如果行不存在则返回空字符串
func getContentLine(lines []string, index int) string {
	if index < len(lines) {
		return lines[index]
	}
	return ""
}

// formatNumberedContent formats content with numbering for each line.
func formatNumberedContent(content string, startCounter int) (string, int) {
	if content == "" {
		return "", startCounter
	}
	lines := strings.Split(content, "\n")
	var formattedLines []string
	counter := startCounter
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			formattedLines = append(formattedLines, fmt.Sprintf("%d. %s；", counter, line))
			counter++
		}
	}
	return strings.Join(formattedLines, "\n"), counter
}

func parsePageBreakMarker(line string) (PageBreak, bool) {
	switch strings.TrimSpace(line) {
	case "{pagebreak}":
		return PageBreak{}, true
	case "{pagebreak:weak}":
		return PageBreak{Weak: true}, true
	default:
		return PageBreak{}, false
	}
}

func startContinuationTable(section *DocumentSection, continuation *tableContinuation, keepCurrentH4 bool) (*Table, *H4Block) {
	if section == nil || continuation == nil {
		return nil, nil
	}

	table := &Table{
		H3Part1: continuation.H3Part1,
		H3Part2: continuation.H3Part2,
	}
	section.Items = append(section.Items, SectionItem{Table: table})

	if !keepCurrentH4 || continuation.H4Title == "" {
		return table, nil
	}

	table.H4Blocks = append(table.H4Blocks, H4Block{Title: continuation.H4Title})
	return table, &table.H4Blocks[len(table.H4Blocks)-1]
}

func tableColumnSpec(table Table) string {
	widths := tableColumnWidthsCM([]Table{table})
	var parts []string
	for _, width := range widths {
		parts = append(parts, fmt.Sprintf("%.2fcm", width))
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

func sectionColumnSpecs(section DocumentSection) []string {
	var chapterTables []Table
	specs := []string{}

	flushChapter := func() {
		if len(chapterTables) == 0 {
			specs = append(specs, tableColumnSpec(Table{}))
			return
		}
		specs = append(specs, tableColumnSpecForTables(chapterTables))
	}

	for _, item := range section.Items {
		if item.PageBreak != nil {
			flushChapter()
			chapterTables = nil
			continue
		}
		if item.Table != nil && len(item.Table.H4Blocks) > 0 {
			chapterTables = append(chapterTables, *item.Table)
		}
	}

	flushChapter()
	return specs
}

func tableColumnSpecForTables(tables []Table) string {
	widths := tableColumnWidthsCM(tables)
	var parts []string
	for _, width := range widths {
		parts = append(parts, fmt.Sprintf("%.2fcm", width))
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

func tableColumnWidthsCM(tables []Table) []float64 {
	headerMetrics := []int{
		displayWidth("教学活动"),
		displayWidth("学习内容"),
		displayWidth("学生活动"),
		displayWidth("教师活动"),
		displayWidth("教学方法与手段"),
		displayWidth("课时分配"),
	}
	metrics := make([]int, len(headerMetrics))
	copy(metrics, headerMetrics)

	for _, table := range tables {
		metrics[0] = maxInt(metrics[0], displayWidth("学习环节"))
		metrics[1] = maxInt(metrics[1], displayWidth(table.H3Part1))
		metrics[2] = maxInt(metrics[2], displayWidth("学习单元"))

		for _, h4 := range table.H4Blocks {
			metrics[0] = maxInt(metrics[0], displayWidth(h4.Title))
			for _, h5 := range h4.H5Blocks {
				metrics[1] = maxInt(metrics[1], displayWidth(getContentLine(h5.Content, 0)))
				metrics[2] = maxInt(metrics[2], displayWidth(getContentLine(h5.Content, 1)))
				metrics[3] = maxInt(metrics[3], displayWidth(getContentLine(h5.Content, 2)))
				metrics[4] = maxInt(metrics[4], displayWidth(getContentLine(h5.Content, 3)))
				metrics[5] = maxInt(metrics[5], displayWidth(h5.Title))
			}
		}
	}

	widths := []float64{
		headerMinWidthCM(maxInt(headerMetrics[0], displayWidth("学习环节")), 0.10),
		headerMinWidthCM(headerMetrics[1], 0.10),
		headerMinWidthCM(headerMetrics[2], 0.10),
		headerMinWidthCM(headerMetrics[3], 0.10),
		headerMinWidthCM(headerMetrics[4], 0.14),
		headerMinWidthCM(headerMetrics[5], 0.10),
	}

	remainingWidth := tableTotalWidthCM
	for _, width := range widths {
		remainingWidth -= width
	}

	if remainingWidth <= 0 {
		return widths
	}

	pressures := columnPressures(tables)
	baseWeights := []float64{0.5, 1.8, 1.6, 1.6, 0.18, 0.06}
	pressureScales := []float64{0.22, 1.0, 0.95, 0.95, 0.18, 0.05}

	totalWeight := 0.0
	weights := make([]float64, len(widths))
	for i := range widths {
		weights[i] = baseWeights[i] + pressureScales[i]*math.Sqrt(pressures[i]+1)
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return widths
	}

	for i := range widths {
		widths[i] += remainingWidth * weights[i] / totalWeight
	}

	return widths
}

func scaleColumnWeight(metric int, divisor, minValue, maxValue float64) float64 {
	return clampFloat(float64(metric)/divisor, minValue, maxValue)
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func displayWidth(s string) int {
	maxWidth := 0
	currentWidth := 0

	for _, r := range s {
		if r == '\n' {
			maxWidth = maxInt(maxWidth, currentWidth)
			currentWidth = 0
			continue
		}

		switch {
		case unicode.IsSpace(r):
			currentWidth++
		case r <= unicode.MaxASCII:
			currentWidth++
		default:
			currentWidth += 2
		}
	}

	return maxInt(maxWidth, currentWidth)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func headerMinWidthCM(metric int, bias float64) float64 {
	return float64(metric)*0.18 + 0.42 + bias
}

func columnPressures(tables []Table) []float64 {
	pressures := []float64{1, 1, 1, 1, 0.5, 0.25}

	for _, table := range tables {
		pressures[0] += contentPressure(table.H3Part1) * 0.2
		pressures[1] += contentPressure(table.H3Part2) * 0.15

		for _, h4 := range table.H4Blocks {
			pressures[0] += contentPressure(h4.Title) * 0.7
			for _, h5 := range h4.H5Blocks {
				pressures[1] += contentPressure(getContentLine(h5.Content, 0)) + 4
				pressures[2] += contentPressure(getContentLine(h5.Content, 1)) + 4
				pressures[3] += contentPressure(getContentLine(h5.Content, 2)) + 4
				pressures[4] += contentPressure(getContentLine(h5.Content, 3)) * 0.45
				pressures[5] += contentPressure(h5.Title) * 0.25
			}
		}
	}

	return pressures
}

func contentPressure(s string) float64 {
	if strings.TrimSpace(s) == "" {
		return 0
	}

	total := 0.0
	for _, line := range strings.Split(s, "\n") {
		total += float64(displayWidth(line))
	}
	return total
}

func cellAlign(col int, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if col <= 2 {
		return "left"
	}
	if col == 3 {
		return "center + horizon"
	}
	return ""
}
