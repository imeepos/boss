# Sphere Boss 电信 BOSS 管理后台  
## 登录 / 注册页《视觉设计提示词》

> **废弃声明（2026 裁定）**：admin 端为封闭账号模型（契约 `docs/contract/domain-map.md` SYS 域），不开放自助注册；账号由上级管理员经 org/account 流程分配公司、权限后开通。本文档中"注册页"相关章节（§3.2 注册形态、§7 差异表注册列、"注册即开通业务运营账号"文案）全部作废，仅登录页规格继续有效。客户侧注册属用户端（REQ-PORT-007），与本文档无关。

> 以下规格依据定稿截图反向标定，设计基准画布为 **1240 × 768 px**。由于截图存在抗锯齿、缩放与图片压缩，颜色及尺寸允许前端在 **±2 px / 轻微色差**范围内校准；Logo、背景、轨道线、广告图等应优先使用设计源文件导出，避免用截图裁切。

---

## 1. 整体设计语言

### 1.1 风格关键词

- **高端政企科技**
- **深海蓝 × 香槟金**
- **全球化、连接、轨道、星球**
- **稳重、可信、专业、克制**
- **未来城市与数字基础设施**
- **商务而非消费互联网**
- **轻奢科技感，不过度炫光**

### 1.2 视觉表达

页面以“**全球业务连接平台**”为核心视觉概念：

- 深蓝代表企业级系统的安全、稳定与专业。
- 金色代表资源、价值、连接与高端服务。
- 星球、环形轨道、光线网络体现全球网络和业务协同。
- 城市天际线、桥梁、光轨体现运营商基础设施与数字城市。
- 登录卡使用大面积白色，保证操作区域清晰、安静、可信。
- 装饰素材只承担品牌氛围，不应抢占表单层级。

### 1.3 气质控制原则

- 避免使用高饱和霓虹蓝、紫色渐变或强烈玻璃拟态。
- 避免过多发光、描边、立体按钮和复杂动效。
- 金色只用于 Logo 环、品牌副标题、链接及少量轨道亮点。
- 核心操作按钮保持深蓝实色，强调可靠而非活泼。
- 页面整体留白充足，表单元素紧凑、秩序明确。

---

## 2. 精确色板

### 2.1 品牌主色

| 用途 | Token 建议 | 色值 |
|---|---|---:|
| 品牌深海蓝 | `brand-navy-900` | `#10203F` |
| 左侧品牌面板背景 | `brand-navy-950` | `#0F1E3B` |
| 主按钮默认色 | `brand-blue-700` | `#273F70` |
| 主按钮悬停色 | `brand-blue-800` | `#1F355F` |
| 主按钮按下色 | `brand-blue-900` | `#192C50` |
| 主按钮禁用色 | `brand-blue-300` | `#9CAAC3` |
| 品牌亮金 | `brand-gold-500` | `#D5A63A` |
| 金色悬停 | `brand-gold-600` | `#BE8D25` |
| 淡金装饰线 | `brand-gold-300` | `#E5C985` |
| 金色低透明背景 | `brand-gold-50` | `#FBF5E8` |

### 2.2 中性色

| 用途 | Token 建议 | 色值 |
|---|---|---:|
| 页面右侧暖白光 | `neutral-warm-50` | `#FFF8EE` |
| 表单面板背景 | `surface-primary` | `#FCFCFD` |
| 输入框背景 | `surface-input` | `#FFFFFF` |
| 主标题文字 | `text-primary` | `#172744` |
| 左侧白色标题 | `text-on-dark` | `#F7F8FC` |
| 正文文字 | `text-secondary` | `#667085` |
| 辅助说明文字 | `text-tertiary` | `#8A96AA` |
| 页脚文字 | `text-muted` | `#A6B1C3` |
| 输入文字 | `text-input` | `#4E5664` |
| 占位文字 | `text-placeholder` | `#737A86` |
| 默认边框 | `border-default` | `#D7DDE7` |
| 悬停边框 | `border-hover` | `#B4BFCE` |
| 聚焦边框 | `border-focus` | `#31568F` |
| 错误色 | `status-danger` | `#D94B4B` |
| 成功色 | `status-success` | `#2F8F63` |

### 2.3 背景与阴影色

```css
--page-bg-left: #102848;
--page-bg-right: #FFF3E3;
--panel-shadow-color: rgba(15, 30, 59, 0.18);
--input-focus-ring: rgba(39, 63, 112, 0.14);
--gold-glow: rgba(213, 166, 58, 0.22);
```

> 页面大背景应以完整背景图实现，不建议只用 CSS 渐变代替；CSS 渐变仅作为图片加载前的兜底色。

---

## 3. 页面版式规格

## 3.1 基准画布

- 设计基准：`1240 × 768 px`
- 页面容器：`width: 100vw; min-height: 100vh`
- 背景图：全屏覆盖，使用 `background-size: cover`
- 背景定位：`center center`
- 主面板在基准画布中的位置约为：
  - 左：`204 px`
  - 上：`109 px`
  - 宽：`820 px`
  - 高：`564 px`

```css
.auth-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background-position: center;
  background-repeat: no-repeat;
  background-size: cover;
}
```

## 3.2 登录/注册主面板

| 项目 | 规格 |
|---|---:|
| 基准宽度 | `820 px` |
| 基准高度 | `564 px` |
| 左侧品牌区宽度 | `382 px` |
| 右侧表单区宽度 | `438 px` |
| 外圆角 | `14 px` |
| 背景 | 左侧深蓝，右侧暖白 |
| 面板阴影 | `0 20px 52px rgba(15,30,59,.18)` |
| 溢出 | `overflow: hidden` |

```css
.auth-panel {
  width: 820px;
  min-height: 564px;
  display: grid;
  grid-template-columns: 382px 438px;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 20px 52px rgba(15, 30, 59, 0.18);
}
```

### 3.3 左侧品牌区

- 背景色：`#0F1E3B`
- 内容水平居中。
- 左侧面板含轨道、金色节点、丝带等装饰，超出内容区部分应裁切。
- 品牌主体内容安全区：
  - 左右内边距：`48–64 px`
  - 顶部内边距：`96 px` 左右
- 广告轮播宽度：`256 px`
- 左侧内容中心轴与面板中心轴一致。

### 3.4 右侧表单区

- 背景：`#FCFCFD`
- 表单内容宽度：`274 px`
- 表单整体水平居中。
- 页脚固定在面板底部视觉区域，不随表单项数量随意上移。
- 登录与注册共用同一右侧容器，仅调整表单整体的纵向起点。

#### 登录页纵向参考

| 元素 | 距面板顶部 |
|---|---:|
| 表单 Logo 顶部 | `142 px` |
| 主标题顶部 | `200 px` |
| 说明文字顶部 | `230 px` |
| 第一输入框顶部 | `267 px` |
| 主按钮顶部 | `356 px` |
| 注册引导顶部 | `411 px` |
| 页脚基线区域 | `535–550 px` |

#### 注册页纵向参考

| 元素 | 距面板顶部 |
|---|---:|
| 线性 Logo 顶部 | `104 px` |
| 主标题顶部 | `160 px` |
| 说明文字顶部 | `190 px` |
| 第一输入框顶部 | `221 px` |
| 主按钮顶部 | `399 px` |
| 登录引导顶部 | `454 px` |
| 页脚基线区域 | `535–550 px` |

---

## 4. 圆角、阴影与间距系统

### 4.1 圆角

| 场景 | 圆角 |
|---|---:|
| 主面板 | `14 px` |
| 广告轮播卡片 | `8 px` |
| 输入框 | `5 px` |
| 主按钮 | `5 px` |
| 小标签/提示 | `4 px` |
| 圆形 Logo / 图标容器 | `50%` |

### 4.2 阴影

```css
--shadow-panel: 0 20px 52px rgba(15, 30, 59, 0.18);
--shadow-card: 0 8px 24px rgba(3, 13, 31, 0.18);
--shadow-focus: 0 0 0 3px rgba(39, 63, 112, 0.14);
--shadow-gold: 0 4px 16px rgba(213, 166, 58, 0.18);
```

- 输入框默认不使用明显阴影。
- 按钮默认不使用强浮起效果。
- Logo 可保留素材自身微弱蓝金辉光，不额外叠加强光晕。

### 4.3 间距基准

使用 `4 px` 基础栅格：

```text
4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64
```

表单重点间距：

| 位置 | 间距 |
|---|---:|
| 主标题与说明文字 | `5–7 px` |
| 说明文字与表单 | `22–24 px` |
| 输入框之间 | `11–12 px` |
| 最后输入框与按钮 | `12 px` |
| 按钮与切换入口 | `18–20 px` |
| 图标与引导文字 | `7–8 px` |

---

## 5. 字体与字重层级

## 5.1 字体族

优先使用系统无衬线字体，避免因网络字体加载造成布局跳动：

```css
font-family:
  "PingFang SC",
  "Microsoft YaHei",
  "Noto Sans SC",
  "Helvetica Neue",
  Arial,
  sans-serif;
```

英文品牌名称可使用：

```css
font-family: "Inter", "Arial", "Helvetica Neue", sans-serif;
```

> “Sphere Boss”最好作为可编辑文本实现；若品牌有专用字标，则使用 SVG。

## 5.2 字体层级

| 层级 | 字号 / 行高 | 字重 | 字间距 | 用途 |
|---|---|---:|---:|---|
| 品牌英文标题 | `28px / 36px` | `700` | `1.2px` | Sphere Boss |
| 品牌中文副标题 | `13px / 20px` | `500` | `4px` | BOSS 综合业务支撑平台 |
| 左侧补充说明 | `12px / 18px` | `500` | `1px` | 注册页运营说明 |
| 表单主标题 | `20px / 28px` | `700` | `0` | 欢迎登录 / 注册账号 |
| 表单说明 | `12px / 18px` | `400` | `0` | 请输入账号信息等 |
| 输入文字 | `12px / 20px` | `400` | `0` | 用户输入内容 |
| 按钮文字 | `14px / 20px` | `600` | `6–10px` | 登录 / 注册 |
| 引导文字 | `12px / 18px` | `400` | `0` | 还没有账号？ |
| 文本链接 | `12px / 18px` | `600` | `0` | 立即注册 / 返回登录 |
| 页脚 | `12px / 18px` | `400` | `.3px` | 平台名称 |

### 排版注意

- 中文按钮文字可设置 `letter-spacing: 8px`，并用等宽视觉居中。
- “Sphere Boss”中的英文单词间保留标准空格。
- 副标题中的英文 “BOSS” 与中文之间保留约 `8 px` 视觉间隔。
- 主标题禁止使用纯黑，统一使用 `#172744`。

---

## 6. 组件规格

## 6.1 输入框

| 属性 | 规格 |
|---|---:|
| 宽度 | `274 px` |
| 高度 | `33–34 px` |
| 圆角 | `5 px` |
| 左右内边距 | `11 px` |
| 字号 | `12 px` |
| 背景 | `#FFFFFF` |
| 默认边框 | `1px solid #D7DDE7` |
| 悬停边框 | `#B4BFCE` |
| 聚焦边框 | `#31568F` |
| 聚焦外环 | `0 0 0 3px rgba(39,63,112,.14)` |

```css
.auth-input {
  width: 100%;
  height: 34px;
  padding: 0 11px;
  border: 1px solid #D7DDE7;
  border-radius: 5px;
  background: #FFFFFF;
  color: #4E5664;
  font-size: 12px;
  outline: none;
  box-sizing: border-box;
}

.auth-input:focus {
  border-color: #31568F;
  box-shadow: 0 0 0 3px rgba(39, 63, 112, 0.14);
}
```

### 输入状态规则

- 错误状态：边框 `#D94B4B`，下方错误信息 `11px / 16px`。
- 禁用状态：背景 `#F3F5F8`，文字 `#A6B1C3`。
- 密码输入建议增加右侧显隐按钮，图标尺寸 `16 px`，但不得挤压输入内容。
- 不依赖 placeholder 代替字段标签；若业务要求更高可访问性，可增加视觉隐藏的 `<label>`。

---

## 6.2 主按钮

| 属性 | 规格 |
|---|---:|
| 宽度 | `274 px` |
| 高度 | `37 px` |
| 圆角 | `5 px` |
| 默认背景 | `#273F70` |
| 文字颜色 | `#FFFFFF` |
| 字号 | `14 px` |
| 字重 | `600` |

```css
.auth-submit {
  width: 100%;
  height: 37px;
  border: 0;
  border-radius: 5px;
  background: #273F70;
  color: #FFFFFF;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 8px;
  cursor: pointer;
}

.auth-submit:hover {
  background: #1F355F;
}

.auth-submit:active {
  background: #192C50;
}

.auth-submit:focus-visible {
  outline: 3px solid rgba(39, 63, 112, 0.22);
  outline-offset: 2px;
}
```

加载状态：

- 文字保持原位或替换为“登录中 / 注册中”。
- Loading 图标尺寸 `14 px`。
- 加载中禁止重复点击。
- 不使用大面积金色按钮，保持主操作深蓝。

---

## 6.3 链接与账号切换入口

- 普通说明文字：`#7C8799`
- 链接颜色：`#C69835` 或 `#D5A63A`
- 字号：`12 px`
- 字重：`600`
- 默认无下划线。
- Hover 时颜色加深至 `#BE8D25`，可增加下划线。
- 图标与文字整体居中显示。

```css
.auth-switch-link {
  color: #C69835;
  font-weight: 600;
  text-decoration: none;
}

.auth-switch-link:hover {
  color: #BE8D25;
  text-decoration: underline;
}
```

安全盾牌图标：

- 尺寸：`12–14 px`
- 颜色：`#49596B`
- 与文字间距：`7 px`
- 统一使用 SVG，不使用 emoji 或字体图标。

---

## 6.4 Logo 规格

页面存在两类 Logo：

### A. 左侧品牌主 Logo

- 蓝色球体 + 金色环形轨道。
- 展示尺寸约：`62 × 62 px`
- 位于品牌英文标题上方。
- Logo 与标题间距：`20–24 px`
- 保留素材原始比例，不拉伸。
- 推荐 SVG；若为复杂光效，可使用 2× PNG/WebP。

### B. 右侧表单 Logo

#### 登录页

- 使用完整蓝金星球 Logo。
- 展示尺寸：`40 × 40 px`
- 与标题间距：`15–18 px`

#### 注册页

- 使用金色线性星球标。
- 展示尺寸：`32 × 32 px`
- 线条颜色建议：`#DFC789`
- 与标题间距：`18–20 px`

### Logo 禁用规则

- 不改变金环角度。
- 不替换为纯蓝或纯黑版本。
- 不叠加描边、投影或白色底盘。
- Logo 四周保留至少 Logo 宽度 `25%` 的安全空间。

---

## 6.5 左侧点缀素材

包含：

- 金色环形轨道
- 轨道节点
- 微弱星点
- 深蓝建筑轮廓
- 底部金蓝丝带
- 光轨与网络流线

使用规则：

1. 装饰轨道位于左侧面板上方及右上局部，不与 Logo 重叠。
2. 金色节点数量保持克制，禁止随机大量生成。
3. 丝带固定在左侧面板底部，允许被广告卡片部分遮挡。
4. 装饰层应设置 `pointer-events: none`。
5. 素材需在面板范围内裁切，不溢出到右侧表单区。
6. 装饰对比度应低于 Logo 和广告图，不影响文字可读性。

推荐图层顺序：

```text
左侧深蓝底色
→ 轨道与星点
→ 城市/丝带装饰
→ 品牌 Logo 与文字
→ 广告轮播卡片
```

---

## 6.6 广告轮播

### 尺寸与外观

| 属性 | 规格 |
|---|---:|
| 宽度 | `256 px` |
| 高度 | `170 px` |
| 比例 | 约 `3:2` |
| 圆角 | `8 px` |
| 背景 | 深蓝图片 |
| 裁切方式 | `object-fit: cover` |
| 阴影 | `0 8px 24px rgba(3,13,31,.18)` |

### 位置

- 水平居中于左侧品牌区。
- 登录页卡片顶部约距总面板顶部 `293 px`。
- 注册页卡片顶部约距总面板顶部 `306 px`。
- 与上方文案至少保持 `28–34 px` 间距。

### 轮播状态指示器

- 位于广告卡右上角，内缩 `10–12 px`。
- 激活项：金色短条 `12 × 3 px`，颜色 `#D5A63A`。
- 非激活项：灰蓝圆点/短条，颜色 `#738096`。
- 项间距：`4 px`。
- 指示器数量建议 `2–4` 个。
- 自动轮播间隔：`5–6 秒`。
- 切换动画：淡入淡出 `300–400 ms`。
- 鼠标悬停时暂停；系统开启“减少动态效果”时关闭自动轮播。
- 不使用夸张横向飞入、3D 翻转或强闪烁。

### 广告图文字规则

- 广告标题建议控制在 `8–12` 个汉字。
- 标题使用白色，辅助文字使用金色。
- 文字放在视觉低干扰区域，并叠加局部深色渐变保证可读性。
- 重要文字禁止紧贴卡片边缘，安全区至少 `20 px`。

---

## 6.7 页脚

内容示例：

```text
Sphere Boss · 综合业务支撑平台
```

规格：

- 字号：`12 px`
- 颜色：`#A6B1C3`
- 字重：`400`
- 水平居中于右侧面板。
- 距面板底部：约 `14–18 px`
- 中英文之间使用间隔点 `·`，前后保留空格。
- 页脚不应参与表单纵向流动，建议绝对定位或使用三段式 Grid 布局。

---

## 7. 登录页与注册页差异

| 项目 | 登录页 | 注册页 |
|---|---|---|
| 右侧 Logo | 蓝金完整星球 | 金色线性星球 |
| 主标题 | 欢迎登录 | 注册账号 |
| 说明文字 | 请输入账号信息进入管理端 | 创建 Sphere Boss 管理端账号 |
| 输入框数量 | 2 个 | 4 个 |
| 按钮文案 | 登录 | 注册 |
| 切换入口 | 还没有账号？立即注册 | 已有账号？返回登录 |
| 左侧补充说明 | 可不显示 | 注册即开通业务运营账号 |
| 表单起始位置 | 相对偏下 | 相对偏上 |
| 广告卡位置 | 约 `293 px` 起 | 约 `306 px` 起 |

注册页字段顺序：

1. 姓名
2. 账号（3–64 位，字母/数字/`-`/`_`/`.`）
3. 密码（不少于 6 位）
4. 确认密码

---

## 8. 素材资产清单及用途映射

建议前端资产目录：

```text
assets/
├─ auth/
│  ├─ bg-auth-city.webp
│  ├─ bg-auth-city@2x.webp
│  ├─ logo-sphere-full.svg
│  ├─ logo-sphere-line-gold.svg
│  ├─ decor-orbit-left.svg
│  ├─ decor-orbit-top.svg
│  ├─ decor-ribbon-bottom.webp
│  ├─ decor-city-silhouette.webp
│  ├─ icon-shield.svg
│  ├─ icon-eye.svg
│  ├─ icon-eye-off.svg
│  ├─ icon-loading.svg
│  └─ banners/
│     ├─ banner-global-network.webp
│     ├─ banner-optical-resource.webp
│     └─ banner-business-support.webp
```

| 资产 | 格式建议 | 用途 |
|---|---|---|
| `bg-auth-city` | WebP/JPG，2× | 全屏城市、桥梁与左右冷暖光背景 |
| `logo-sphere-full` | SVG | 左侧品牌区、登录页表单顶部 |
| `logo-sphere-line-gold` | SVG | 注册页表单顶部 |
| `decor-orbit-left` | SVG | 页面左上角全局轨道装饰 |
| `decor-orbit-top` | SVG | 左侧品牌面板顶部轨道与节点 |
| `decor-ribbon-bottom` | 透明 WebP/PNG | 左侧面板底部金蓝丝带 |
| `decor-city-silhouette` | 透明 WebP/PNG | 左侧面板底部建筑轮廓 |
| `icon-shield` | SVG | 登录/注册切换提示前图标 |
| `icon-eye` / `eye-off` | SVG | 密码显示隐藏 |
| `icon-loading` | SVG | 提交加载状态 |
| `banner-*` | WebP，建议 512×340 | 广告轮播图 |

### 导出规范

- 位图至少导出 `2×`，避免高分屏模糊。
- 背景图建议控制在 `500 KB` 左右，单张广告图建议小于 `150 KB`。
- SVG 清除编辑器冗余信息，保留 `viewBox`。
- 装饰位图必须透明背景。
- 所有 Logo 与轨道素材禁止从截图中抠图。

---

## 9. 响应式使用规则

### ≥ 1200 px

- 使用完整双栏面板。
- 面板优先保持 `820 × 564 px`。
- 页面四周最小安全间距：`32 px`。

### 960–1199 px

- 面板可按比例缩放或调整为：
  - 总宽度：`760–800 px`
  - 左右比例仍约为 `46.5% : 53.5%`
- 表单宽度保持不低于 `274 px`。
- 不通过压缩输入框高度解决空间问题。

### 768–959 px

- 可隐藏左侧部分装饰，但保留品牌区和广告卡。
- 面板宽度设置为 `calc(100vw - 48px)`。
- 左侧广告可缩小至 `220 × 146 px`。

### < 768 px

- 切换为单栏表单卡片。
- 左侧品牌区隐藏，或缩为顶部品牌横幅。
- 卡片宽度：`calc(100vw - 32px)`，最大 `420 px`。
- 表单宽度：`100%`。
- 卡片内边距：`32 px 24 px 24 px`。
- 保留右侧 Logo、标题、表单和页脚。
- 移动端输入框及按钮高度提升至至少 `44 px`，满足触控要求。
- 背景图保留，但增加轻微暖白/深蓝蒙层确保卡片突出。

---

## 10. 可复用于其它页面的 Design Token

```css
:root {
  /* Brand */
  --color-brand-navy-950: #0F1E3B;
  --color-brand-navy-900: #10203F;
  --color-brand-blue-900: #192C50;
  --color-brand-blue-800: #1F355F;
  --color-brand-blue-700: #273F70;
  --color-brand-blue-300: #9CAAC3;

  --color-brand-gold-600: #BE8D25;
  --color-brand-gold-500: #D5A63A;
  --color-brand-gold-300: #E5C985;
  --color-brand-gold-50: #FBF5E8;

  /* Surface */
  --color-bg-page: #F4F5F7;
  --color-surface-primary: #FCFCFD;
  --color-surface-input: #FFFFFF;
  --color-surface-disabled: #F3F5F8;
  --color-surface-dark: #0F1E3B;

  /* Text */
  --color-text-primary: #172744;
  --color-text-secondary: #667085;
  --color-text-tertiary: #8A96AA;
  --color-text-muted: #A6B1C3;
  --color-text-placeholder: #737A86;
  --color-text-on-dark: #F7F8FC;
  --color-text-link: #C69835;

  /* Border */
  --color-border-default: #D7DDE7;
  --color-border-hover: #B4BFCE;
  --color-border-focus: #31568F;
  --color-border-error: #D94B4B;

  /* Status */
  --color-success: #2F8F63;
  --color-warning: #D5A63A;
  --color-danger: #D94B4B;
  --color-info: #31568F;

  /* Typography */
  --font-family-base:
    "PingFang SC",
    "Microsoft YaHei",
    "Noto Sans SC",
    "Helvetica Neue",
    Arial,
    sans-serif;

  --font-family-brand:
    "Inter",
    "Arial",
    "Helvetica Neue",
    sans-serif;

  --font-size-xs: 11px;
  --font-size-sm: 12px;
  --font-size-md: 14px;
  --font-size-lg: 16px;
  --font-size-xl: 20px;
  --font-size-brand: 28px;

  --font-weight-regular: 400;
  --font-weight-medium: 500;
  --font-weight-semibold: 600;
  --font-weight-bold: 700;

  /* Radius */
  --radius-xs: 4px;
  --radius-sm: 5px;
  --radius-md: 8px;
  --radius-lg: 14px;
  --radius-round: 999px;

  /* Spacing */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 20px;
  --space-6: 24px;
  --space-8: 32px;
  --space-10: 40px;
  --space-12: 48px;
  --space-16: 64px;

  /* Control */
  --control-height-sm: 34px;
  --control-height-md: 37px;
  --control-height-touch: 44px;

  /* Shadow */
  --shadow-panel: 0 20px 52px rgba(15, 30, 59, 0.18);
  --shadow-card: 0 8px 24px rgba(3, 13, 31, 0.18);
  --shadow-focus: 0 0 0 3px rgba(39, 63, 112, 0.14);

  /* Motion */
  --motion-fast: 160ms;
  --motion-normal: 240ms;
  --motion-carousel: 360ms;
  --ease-standard: cubic-bezier(0.2, 0, 0, 1);
}
```

---

## 11. 前端实现验收重点

1. 主面板在 `1240 × 768 px` 下视觉尺寸约为 `820 × 564 px`，且居中。
2. 左右面板比例保持约 `382 : 438`。
3. 右侧表单有效宽度统一为 `274 px`。
4. 登录、注册页不能简单共用同一垂直起点，应分别匹配截图。
5. 背景图不可拉伸变形，必须使用 `cover`。
6. 左侧装饰必须裁切在深蓝面板内部。
7. Logo、轨道、盾牌图标必须使用独立高清资产。
8. 链接使用金色，主按钮使用深蓝，禁止互换。
9. 输入框需具备 Hover、Focus、Error、Disabled 状态。
10. 广告轮播不应造成布局位移，并支持减少动态效果。
11. 所有可交互元素需有键盘焦点状态。
12. 移动端触控组件高度不低于 `44 px`。