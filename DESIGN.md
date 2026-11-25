---
name: RelayRadar
description: 独立监测大模型 API 中转站的实时仪表台
colors:
  signal-green: "#2ee6a6"
  deep-charcoal: "#06070a"
  surface: "#0b0e13"
  surface-raised: "#11151c"
  hairline: "#1a1f28"
  hairline-strong: "#232b37"
  ink: "#e9edf4"
  ink-muted: "#a7b1c2"
  ink-faint: "#7a8599"
  warning-amber: "#f5b942"
  outage-red: "#ff5d5d"
  info-blue: "#6aa8ff"
typography:
  display:
    fontFamily: "Inter, Segoe UI Variable Display, Segoe UI, PingFang SC, Microsoft YaHei, sans-serif"
    fontSize: "clamp(46px, 4.4vw, 62px)"
    fontWeight: 850
    lineHeight: 1.02
    letterSpacing: "-0.045em"
  headline:
    fontFamily: "Inter, Segoe UI Variable Display, Segoe UI, PingFang SC, Microsoft YaHei, sans-serif"
    fontSize: "24px"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-0.02em"
  body:
    fontFamily: "Inter, Segoe UI Variable Text, Segoe UI, PingFang SC, Microsoft YaHei, sans-serif"
    fontSize: "14.5px"
    fontWeight: 400
    lineHeight: 1.62
  label:
    fontFamily: "JetBrains Mono, Cascadia Code, SFMono-Regular, Consolas, monospace"
    fontSize: "10.5px"
    fontWeight: 500
    letterSpacing: "0.14em"
rounded:
  sm: "8px"
  md: "12px"
  lg: "16px"
spacing:
  xs: "8px"
  sm: "12px"
  md: "16px"
  lg: "24px"
  xl: "32px"
  section: "56px"
components:
  button-primary:
    backgroundColor: "{colors.signal-green}"
    textColor: "{colors.deep-charcoal}"
    rounded: "{rounded.sm}"
    padding: "0 18px"
    height: "38px"
  button-primary-hover:
    backgroundColor: "#5ff0bd"
    textColor: "{colors.deep-charcoal}"
    rounded: "{rounded.sm}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.sm}"
    padding: "0 16px"
    height: "38px"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.ink}"
    rounded: "{rounded.lg}"
    padding: "22px 24px"
  input:
    backgroundColor: "{colors.deep-charcoal}"
    textColor: "{colors.ink}"
    rounded: "{rounded.sm}"
    padding: "8px 12px"
    height: "38px"
---

# Design System: RelayRadar

## Overview

**Creative North Star: "监测仪表台"**

RelayRadar 看起来像一台持续运行的监测仪表：深炭底色压住画面，荧光绿只标记“信号正常”与主行动，数据以等宽字体成列排布。页面不追求营销页的装饰，而追求读数清晰、层级可扫、状态一眼可辨。

信息密度高，但靠留白分组：标题上方留白大于下方，卡片只包裹真正成组的内容，其余用细线和编辑式分栏。动效克制，只服务于“数据在更新”和“状态变化”。

**Key Characteristics:**
- 深炭底、单一信号绿、三色状态（绿/琥珀/红）
- 数据用等宽字体、表格数字对齐
- 扁平为默认，浮起只属于 hover、浮层与告警
- 中文优先，中文标题比英文标签重

## Colors

深炭背景上的信号绿，像仪表盘上的工作指示灯。

### Primary
- **信号绿** (#2ee6a6)：主按钮、关键数字、正常状态、可点击强调。任何一屏不超过约 10%。

### Neutral
- **深炭** (#06070a)：页面底色。
- **面板** (#0b0e13)：内容容器。
- **浮起面板** (#11151c)：导航 hover、次级表面。
- **细线** (#1a1f28)：分隔与边框；更强一级用 #232b37。
- **墨色** (#e9edf4)：主文本。
- **次墨** (#a7b1c2)：正文与说明。
- **弱墨** (#7a8599)：元信息、单位、时间（面板上 5.19:1）。

### Status
- **琥珀** (#f5b942)：降级、波动、注意事项。
- **告警红** (#ff5d5d)：中断、不可达、危险。
- **信息蓝** (#6aa8ff)：中性数据与链接辅助。

### Named Rules
**The One Signal Rule.** 信号绿是“正常/主行动”的专用色；不要用它装饰标题、边框或背景。
**The Status Only Rule.** 琥珀与红只表达状态，不用于品牌强调。

## Typography

**Display Font:** Inter / Segoe UI Variable（中文回退 PingFang SC / 微软雅黑）
**Body Font:** 同上
**Label/Mono Font:** JetBrains Mono / Cascadia Code（数据与标签）

**Character:** 标题厚重、字距收紧；数据等宽、成列对齐；标签小写字母间距拉开，像仪表刻度。

### Hierarchy
- **Display** (850, clamp(46–62px), 1.02)：仅首页 Hero。
- **Headline** (700, 24–26px, 1.25)：区块标题。
- **Title** (650, 16–18px)：卡片与面板标题。
- **Body** (400, 14.5px, 1.62)：正文与说明，行宽不超过 75ch。
- **Label** (500, 10.5px, 0.14em, 大写)：表头、单位、状态标签。

### Named Rules
**The Tabular Rule.** 所有数字列使用 `font-variant-numeric: tabular-nums`，榜单不因刷新抖动。
**The No Gradient Text Rule.** 强调靠字重、字号与颜色，不用渐变文字。

## Layout

容器最大 1320px（内容页 1240px），两侧 24–32px 内边距。首页 Hero 为 1.15:0.85 双栏，桌面间距 64px；≤980px 单列。区块之间 56px，卡片内部 16–24px。移动端卡片列表替代宽表，工具栏换行不横向滚动。

## Elevation & Depth

扁平为默认。面板只用 1px 细线与轻微内高光分层；阴影只在 hover、下拉、弹层和告警浮层出现，且带偏移与柔和模糊，不做零偏移光晕。

## Shapes

圆角三档：8px 控件、12px 小卡、16–22px 面板与 Hero 卡。边框始终 1px；不使用单侧装饰条，也不用硬偏移阴影。

## Components

### Buttons
- **Shape:** 8px 圆角，38px 高。
- **Primary:** 信号绿底、深炭字，0 18px 内边距。
- **Hover / Focus:** hover 提亮到 #5ff0bd；focus-visible 2px 绿色外环。
- **Ghost:** 透明底、次墨字、1px 细线。

### Chips
- **Style:** 透明底、1px 细线、弱墨字；选中时绿色边框与 7% 绿色底。

### Cards / Containers
- **Corner Style:** 12–16px。
- **Background:** #0b0e13 到 #090d12 的轻微渐变。
- **Border:** 1px #1a1f28。
- **Internal Padding:** 16–24px。

### Inputs / Fields
- **Style:** 深炭底、1px 细线、8px 圆角、38px 高。
- **Focus:** 边框转信号绿并保留 2px 外环。

### Navigation
- 顶部导航 13.5px，默认次墨；hover 浮起面板底；active 绿色文字 + 12% 绿底。移动端折叠为抽屉。

## Do's and Don'ts

### Do:
- **Do** 让信号绿只出现在主行动、正常状态和关键数字上。
- **Do** 用等宽数字与表格对齐承载比较。
- **Do** 让标题上方留白大于下方。
- **Do** 用中文承担主要信息，英文只做辅助标签。

### Don't:
- **Don't** 用渐变文字、光晕或玻璃模糊做装饰。
- **Don't** 用单侧彩色边条装饰卡片。
- **Don't** 用大数字 + 小标签的营销 Hero 替代真实数据。
- **Don't** 用琥珀或红色表达非状态信息。
