// 中文字号转换函数
#import "@preview/pointless-size:0.1.2": zh

// 定义常用字体名称
#let FONT_XBS = "FZXiaoBiaoSong-B05" // 方正小标宋
#let FONT_HEI = "STHeiti" // 黑体
#let FONT_FS = "STFangsong" // 仿宋
#let FONT_KAI = "STKaiti" // 楷体
#let FONT_SONG = "STSong" // 宋体

// 设置页面、页边距、页脚
#set page(
  paper: "a4",
  margin: (
    inside: 28mm,
    outside: 26mm,
    top: 37mm,
    bottom: 35mm,
  ),

  // 将页脚基线放到"版心下边缘之下 7mm"
  footer-descent: 7mm,

  // 使用更稳定的奇偶页判断和页码格式
  footer: context {
    let page-num = here().page()
    let is-even = calc.even(page-num)
    let num = str(page-num)
    let pm = text(font: FONT_SONG, size: zh(4))[— #num —] // 4 号宋体

    if is-even {
      align(left, [#h(1em) #pm]) // 偶数页：居左
    } else {
      align(right, [#pm #h(1em)]) // 奇数页：居右
    }
  },
)

// 设置文档默认语言和正文字体
#set text(
  lang: "zh",
  font: FONT_FS,
  size: zh(3),
  hyphenate: false,
  cjk-latin-spacing: auto,
)

// 设置段落样式，以满足"每行28字符，每页22行"的网格标准，首行缩进2字符
#set par(
  first-line-indent: (amount: 2em, all: true),
  justify: true,
  leading: 15.6pt, // 行间距
  spacing: 15.6pt, // 段间距
)

// 计数器设置
#let h2-counter = counter("h2")
#let h3-counter = counter("h3")
#let h4-counter = counter("h4")
#let h5-counter = counter("h5")

// 图片样式设置
#show figure: it => {
  // 居中对齐，无首行缩进
  set par(first-line-indent: 0pt)
  align(center, block({
    // 图片尺寸由 Lua filter 控制
    it.body

    // 图注样式：3号仿宋，格式为"图1 标题"
    text(
      font: FONT_FS,
      size: zh(3),
      it.caption,
    )
  }))
}

// 自定义标题函数
#let custom-heading(level, body, numbering: auto) = {
  if level == 1 {
    v(0pt)
    align(center)[
      #text(
        font: FONT_XBS,
        size: zh(2),
        weight: "bold",
      )[
        #set par(leading: 35pt - zh(2))
        #body
      ]
    ]
    v(28.7pt)
  } else if level == 2 {
    h2-counter.step()
    h3-counter.update(0)
    h4-counter.update(1)
    h5-counter.update(1)
    text(
      font: FONT_HEI,
      size: zh(3),
    )[#context h2-counter.display("一、")#body]
  } else if level == 3 {
    h3-counter.step()
    h4-counter.update(1)
    h5-counter.update(1)

    let number = h3-counter.get().first()
    text(
      font: FONT_KAI,
      size: zh(3),
    )[#context h3-counter.display("（一）")#body]
  } else if level == 4 {
    h4-counter.step()
    h5-counter.update(1)

    let number = h4-counter.get().first()
    text(
      size: zh(3),
    )[#number. #body]
  } else if level == 5 {
    h5-counter.step()

    let number = h5-counter.get().first()
    text(
      size: zh(3),
    )[（#number）#body]
  }
}

#show heading: it => {
  if it.level == 1 {
    custom-heading(it.level, it.body, numbering: it.numbering)
  } else {
    let spacing = 13.9pt
    let threshold = 3em

    block(
      sticky: true,
      above: spacing,
      below: spacing,
      {
        block(
          custom-heading(it.level, it.body, numbering: it.numbering) + v(threshold),
          breakable: false,
        )
        v(-threshold)
      },
    )
  }
}

#h2-counter.update(0)
#h3-counter.update(0)
#h4-counter.update(0)
#h5-counter.update(0)

#let list-depth = state("list-depth", 0)

#let flush-left-list(it) = {
  list-depth.update(d => d + 1)

  let is-enum = (it.func() == enum)
  let children = it.children

  context {
    let depth = list-depth.get()
    let block-indent = if depth > 1 { 2em } else { 0pt }

    pad(left: block-indent, block({
      for (count, item) in children.enumerate(start: 1) {
        if item.func() == list.item or item.func() == enum.item {
          let marker = if is-enum {
            let pattern = if it.has("numbering") and it.numbering != auto { it.numbering } else { "1." }
            numbering(pattern, count)
          } else {
            if it.has("marker") and it.marker.len() > 0 { it.marker.at(0) } else { [•] }
          }

          par(
            first-line-indent: par.first-line-indent,
            hanging-indent: 0pt,
          )[#marker#h(0.25em)#item.body]
        } else {
          item
        }
      }
    }))

    list-depth.update(d => d - 1)
  }
}

#show list: flush-left-list
#show enum: flush-left-list

#let name(name) = align(center, pad(bottom: 0.8em)[
  #text(font: FONT_KAI, size: zh(3))[#name]
])
