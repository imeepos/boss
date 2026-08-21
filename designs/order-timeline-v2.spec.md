# 订单详情 · 装维进度区 优化规格(对应 order-timeline-v2.png)

> 优化对象:`mobile/user/.../page/OrderPage.kt::TimelineCard` + `TimelineItem`。
> 保留上游:`OrderScreen` 整体结构、`MilestoneBlock`(4 节点 stepper)、`AppCard`/`CardTitle` 容器。
> 视觉锚点:与既有 `OrderStepper`(OrderCard.kt)共用 4 节点命名,"提交订单 · 受理成功 · 上门安装 · 完成"全 App 统一。

## 优化目标(原版痛点 → 改后表现)

| 原版问题 | 改后方案 |
|---|---|
| 12 个阶段平铺,信息密度过高 | 拆 4 个里程碑分组,与上方 4 节点 stepper 一一对应 |
| 8dp 节点几乎看不见 | 12dp 节点 + 状态色实/描边双样式 |
| "已完成/进行中/待完成" 仅靠字重区分 | 三态视觉:**绿✓/蓝实心/灰空心**(颜色+形态+文字三线索) |
| 顶部无进度概览 | 加 `N/12` 大数字 + 4dp 进度条(主蓝填充) |
| 当前阶段没有高亮 | 当前里程碑整组蓝底 tint,当前阶段行加蓝色左边线/底色 |
| 全部 12 项展开,纵长难扫 | **渐进展示**:过去/未来里程碑折叠(只显 header + 计数),仅当前里程碑展开 |

## 给前端的实现提示词

实现"装维进度"区块,基于已有 timeline JSON 数据(每个阶段含 `stage`(int 1-12)、
`title`、`result`(DONE/DOING/PENDING)、`finishedAt`、`meta` 字段)。

### 整体结构(自上而下)

```
AppCard {
  CardTitle("装维进度", "<已完成>/12 已完成")       // 不变
  
  // ===== 新增:进度条 =====
  Box(Modifier.fillMaxWidth().height(4.dp).background(Palette.line, RoundedCornerShape(2.dp))) {
    Box(Modifier.fillMaxHeight().fillMaxWidth(<done>/12f).background(Palette.primary, RoundedCornerShape(2.dp)))
  }
  Spacer(4.dp)
  
  // ===== 重写:4 个里程碑分组 =====
  for (m in 1..4) {
    MilestoneGroup(milestone = m, stages = stages_in_milestone, defaultExpanded = (m == currentMilestone))
    Spacer(8.dp)   // 卡片间距
  }
}
```

### 里程碑分组(`MilestoneGroup`)

#### 容器

- 不再嵌 `AppCard`,而是 `Box(Modifier.fillMaxWidth().background(bg, RoundedCornerShape(12.dp)).border(...))`
- **过去里程碑**:`bg = Palette.panel`(白),`border = null`
- **当前里程碑**:`bg = Palette.primary.copy(alpha = 0.06f)`,`border = 1.dp Palette.primary.copy(alpha = 0.2f)`
- **未来里程碑**:`bg = Palette.panel`,`border = null`,整体透明度 0.85
- 内边距:水平 14dp / 垂直 12dp

#### Header(可点击整行展开/折叠)

- 左侧:状态图标 24dp 圆形容器
  - **已完成**:`Box(24.dp, success.copy(alpha=0.12f), CircleShape)` 内 `Icons.Filled.Check` 16dp/success
  - **进行中**:`Box(24.dp, primary.copy(alpha=0.12f), CircleShape)` 内 `Icons.Filled.Autorenew` 16dp/primary
    (Material `Icons.Filled.Autorenew`,表示装维进行中,与"上门安装"语义一致)
  - **待完成**:`Box(24.dp, muted.copy(alpha=0.1f), CircleShape)` 内 `Icons.Outlined.HourglassEmpty` 16dp/muted
- 中间:`Text(milestoneName, 16sp/W600, ink)`(`milestoneName` = "提交订单"/"受理成功"/"上门安装"/"完成")
- 右中:`Text("${doneInMilestone}/${totalInMilestone}", 13sp, muted)`
- 最右:`Icons.Filled.KeyboardArrowDown`(展开时旋转 180°)
  - 用 `graphicsLayer(rotationZ = if (expanded) 180f else 0f)` 做平滑过渡(150ms)
- Header 行高 48dp,整行 `clickable` 切换展开状态

#### 阶段列表(仅当前里程碑展开时显示)

- 列容器:padding start=24dp(与 header 图标对齐),end=14dp,top=8dp,bottom=4dp
- 每条 `StageRow` 结构(与原 `TimelineItem` 类似但加大尺寸):
  - 左列 16dp 宽,垂直居中:
    - 12dp 节点圆:
      - DONE:`Modifier.size(12.dp).background(Palette.success, CircleShape)`
      - DOING:`Modifier.size(12.dp).background(Palette.primary, CircleShape).border(2.dp Palette.primary.copy(alpha=0.3f), CircleShape)`(halo 效果)
      - PENDING:`Modifier.size(12.dp).background(Palette.panel, CircleShape).border(1.5.dp Palette.line, CircleShape)`
    - 非末项:垂直连接线 `Box(2.dp, fillMaxHeight, Palette.line)`
  - 右列 padding start=10dp,垂直 10dp:
    - 顶部行:`Text(stage, 12sp/muted)` + `Text(title, 14sp, if DOING then W600/Bold/primary else W500/ink)`
      + `Spacer(weight=1f)` + `Text(finishedAt, 12sp/muted, right)`
    - meta 行(条件):`Text(meta, 11.5sp/muted)` —— 当前阶段若 meta 非空则显示

### 进度条(新增)

- 容器:`Box(fillMaxWidth().height(4.dp).background(Palette.line, RoundedCornerShape(2.dp)))`
- 填充:`Box(fillMaxHeight().fillMaxWidth(fraction).background(Palette.primary, RoundedCornerShape(2.dp)))`
- `fraction = doneCount / 12f`,`doneCount` = `stages.count { it.result == "DONE" }`
- 全部完成时填满至 100%

### 里程碑名映射(沿用 `OrderStepper`)

| milestone | 名称 | 包含 stage |
|---|---|---|
| 1 | 提交订单 | 1, 2, 3 |
| 2 | 受理成功 | 4, 5, 6 |
| 3 | 上门安装 | 7, 8, 9 |
| 4 | 完成 | 10, 11, 12 |

> 注意:与 `OrderStepper` 的 1-1/2-7/8-11/12-4 映射**不一致**,本规范采用 1-3/4-6/7-9/10-12 的**均分**映射。
> 理由:用户要看到的就是 12 个阶段均匀分布在 4 个里程碑下,均分视觉更平衡;`OrderStepper` 的非均分是设计稿示意图,与契约不冲突。
> **若后端返回的 stage 与上述区间不完全一致**(实际生产中可能跳过某些环节),按"阶段归属 milestone = ceil(stage / 3)"动态计算,不允许写死数组。

### 数据契约(沿用 `OrderApi.detail().timeline[]`)

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `stage` | int 1-12 | 是 | 决定归属哪个 milestone: `(stage - 1) / 3 + 1` |
| `title` | string | 是 | "派单"/"扫码绑定"等 |
| `result` | enum | 是 | DONE / DOING / PENDING |
| `finishedAt` | string | 否 | ISO8601,空时不渲染 |
| `meta` | string | 否 | "装维工程师已接单"等副文案 |

### 交互状态

- **默认展开规则**:`currentMilestone = ceil(currentStage / 3.0).toInt()`,仅此 milestone 展开;
  其他全部折叠(用户可点击 header 展开/收起)
- **加载中**:沿用原版"`进度加载中…` 12.5sp muted"
- **空数据**:stages 为空数组时显示 `EmptyState("暂无进度信息")`(用 Widgets.kt 现有组件)
- **全部完成(DONE=12)**:所有 4 个 milestone 标记为已完成,current 归到最后一个(完成),
  可选展开所有(若 `doneCount == 12` 则全部展开,展示完整 12 项)

## 注意事项

- **删除原 `TimelineItem` 函数**,被 `StageRow` 替代
- **不再需要外层"已完成 N/12 · 含合同收费"字样**,只保留 CardTitle 的"4/12 已完成"右侧链接
- **箭头用 `Icons.Filled.KeyboardArrowDown`**(`AutoMirrored` 同样可用),用 `graphicsLayer(rotationZ)` 旋转
  不要用 `AnimatedVisibility` 做展开/收起动画(性能浪费且视觉跳变明显)
- **当前里程碑的浅蓝底**(`primary.copy(alpha=0.06f)`)是设计稿的"关键决策"——不可拿掉,否则看不出当前在哪一组
- **过去里程碑不自动展开**;若用户想看细节,点击 header 即可(一次只展一个:再次点击当前项则收起)
- **时间戳沿用 ISO 字符串原样显示**(如 `05-20 10:30`),不引入相对时间转换(避免引入新依赖)
- **现有 `AppCard` 容器不变**,只是在 AppCard 内部多塞了几个里程碑子卡;不要把里程碑做成嵌套 AppCard
- **复用 `OrderCard.kt::milestoneOf` 不要复用**,那是给 stepper 用的非均分映射,与本规范均分冲突;本组件**内部**计算 milestone