# 套餐详情页 实现规格（对应 product-detail-v1.png）

> 对应代码:`mobile/user/android/app/src/main/java/com/ymm/boss/user/page/ProductPage.kt`
> 数据契约:`api/openapi/user/product.yaml::/products/{productId}` → `ProductDetail { product, specs, compare }`
> 路由:`Route.Product(id: String)`
> 视觉规范:`designs/USER-APP-SPEC.md`(以本规范为准,UI-SPEC.md 为历史参考)

## 给前端的实现提示词（可直接复制）

实现 Android Compose 的套餐详情页,高保真还原附图 `product-detail-v1.png`。
沿用现有 `Widgets.kt` 的 `TopBar` / `AppCard` / `CardTitle` / `CellRow` / `PricePill` / `IconTile` / `EmptyState` /
`Notice`,禁止重造。

- **整体布局**:`Column + verticalScroll(rememberScrollState())` 包裹,底部固定一行 `Surface(tonalElevation=2.dp)`
  装 CTA 按钮;滚动区与底栏之间留 14dp。
- **状态栏**:固定纯色 `#006AE5`(`statusBarSolid()`),与 TopBar 同源消除接缝。
- **TopBar**:`TopBar("套餐详情", onBack = { nav.pop() })`,48dp 高,蓝底白字。
- **色彩**:主色 `Palette.primary #007AFF`、页面底 `Palette.bg #F5F6F8`、卡片白 `Palette.panel #FFFFFF`、
  分割线 `Palette.line #E8EAED`、主文字 `Palette.ink #1C1C1E`、辅助 `Palette.muted #3F3F46`、
  热门绿 `Palette.success #34C759`、价格橙 `Palette.warn #FF9500`、错误红 `Palette.err #FF3B30`。
- **字体层级**:
  - 产品名 15sp/W600、描述 12sp/16sp muted;
  - 卡片标题 15sp/W600 ink + 12.5sp 链接 muted(`CardTitle`);
  - 规格 label 14sp/W500 ink,value 13sp muted(右对齐);
  - 对比行产品名 14sp/W500 ink(当前套餐 14sp/W600 primary);
  - 当前套餐徽章 11sp 白字 + `primary` 实底 4dp 圆角;
  - 合约说明 12.5sp muted(`Notice`)。
- **数据区块**(自上而下):
  1. **主信息卡**(`AppCard`):`Row` = 名称(`weight=1f`)+ 条件热门 `Tag("热门", Palette.success)` + `PricePill(monthlyFee)`;
     下方 `Notice(description)`。产品名为空时显示"加载中…",月费为空时显示"—"。
  2. **套餐规格卡**(`AppCard` + `CardTitle("套餐规格")`):遍历 `specs[]`,
     每项 = `CellRow(title=label, right={Text(value)})`,无 specs 时不渲染该卡。
  3. **套餐对比卡**(`AppCard` + `CardTitle("套餐对比")`):遍历 `compare[]`,
     每行 = 名称(W500 ink,若是当前 id 则 W600 primary)+ 12sp muted 描述 + 右侧 `PricePill(monthlyFee)` +
     `KeyboardArrowRight 20dp muted`。当前行名称后追加"当前套餐"小 Tag(`primary` 实底白字 11sp,水平 6dp 垂直 2dp 内边距,4dp 圆角)。
     无可比项时显示 `EmptyState("暂无可比套餐")`,有则行可点击 `nav.push(Route.Product(productId))`。
  4. **合约说明卡**(`AppCard` + `CardTitle("合约与说明")`):`Notice("合约期内退订按未履约月份收取违约金;改套餐当月按新旧价按日折算(pro-rata)。")`。
  5. **底部 CTA 栏**:`Surface` 满宽,2dp tonalElevation,白底,内边距 `horizontal=14dp, vertical=10dp`;
     `Button(containerColor=Palette.primary, shape=RoundedCornerShape(8.dp), Modifier.fillMaxWidth().height(44.dp))`
     → `Text("立即办理 ¥{monthlyFee}/月", fontSize=14sp, fontWeight=W500, color=White)`。
     点击 → `OrderApi.submit(id, DEMO_ADDRESS_ID)` → 成功后 `nav.push(Route.Order(orderNo))`;失败 `Notice(err, Palette.err)`。
- **交互状态**:
  - **加载**:`product == null` 时各卡显示占位文字("加载中…"/"—"),不渲染骨架屏。
  - **错误**:接口异常 → 卡上方 `Notice("套餐详情加载失败", Palette.err)`。
  - **空**:specs/compare 为空 → 不渲染该卡或 `EmptyState("暂无可比套餐")`。
- **数据接口**:
  - `ProductApi.detail(id)` 返回 `{product:{productId, name, bandwidth, monthlyFee, contractMonths, featured, description}, specs:[{label,value}], compare:[Product]}`。
  - 当前套餐判断:`p.optString("productId") == id`。
  - `DEMO_ADDRESS_ID = "ADDR-001"`(占位,后续接地址选择路由时替换)。

## 注意事项

- **图标**:主信息卡目前无图标;若需美化可加 44dp Router 浅蓝 IconTile,与 ProductsPage 产品卡一致。规格卡标题左侧可加 14dp Description 主色线性图标(纯装饰,可省)。
- **价格胶囊复用**:`Widgets.kt::PricePill` 已实现,`recommended=false` 即普通档蓝浅底;
  若当前套餐是推荐档(代码里 `bandwidth.contains("500")` 判定),传 `recommended=true` 得橙底白字。
  本页 `DetailCard` 默认不强调推荐,使用普通档;`CompareCard` 中非当前行沿用普通档,当前行使用普通档(已在主信息卡展示)。
- **不在 spec 自行添加**:`Product` 字段表无 `originalPrice`/`features[]`(USER-APP-SPEC §11),不要画划线价或卖点图标列表。
- **响应式**:仅竖屏,375dp 设计宽度基准;Compose 自然适配。
- **可访问性**:`IconButton`/ `Button` 自带 44dp+ 触控热区,价格胶囊/Tag 为展示性元素无需 focus;
  对比度:主文字 on 白 ≥10:1、辅助文字 ≥7:1,均通过 WCAG AA。
- **性能**:规格行数固定(<10),不虚拟化;对比列表固定 3 项,无需 LazyColumn;滚动用 `Column + verticalScroll` 即可。
- **iOS 风状态栏**:`statusBarSolid()` 已统一为 `#006AE5`,不要在 `MainActivity` 单独改 Window statusBarColor。