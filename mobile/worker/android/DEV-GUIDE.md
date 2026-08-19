# 师傅端 Android 页面开发指引（子代理必读）

> 目标：把 `docs/worker/*.html`（H5 草稿）逐个实现为 Compose 页面。
> 本文是子代理的契约包：只写你自己的页面文件，**禁止修改共享文件**（Nav.kt / AppRoot.kt / Widgets.kt / Load.kt / Api.kt / WorkerApi.kt / MainActivity.kt）。

## 目录结构

- 页面文件：`mobile/worker/android/app/src/main/java/com/ymm/boss/worker/ui/<ScreenName>Screen.kt`
- 每个页面一个文件，单文件 ≤ 300 行，单函数 ≤ 30 行
- 包名 `com.ymm.boss.worker.ui`
- 页面 Composable 命名约定：`<ScreenName>Screen(nav: NavHost, ...)`，与 Screen 密封类条目对应

## 路由（Nav.kt 已定义，禁止修改）

```kotlin
sealed interface Screen {
    data object Login/Home/Orders/Profile : Screen
    data class TicketDetail(no) / Repair(no) / Scan(no) / Photo(no) / ScanAbnormal(no) /
        Report(no) / Activate(no) / Charge(no) / Sign(no) / Checkin(no) / Navi(no) /
        Transfer(no) / Reschedule(no) / Complaint(no) / Dismantle(no) / Replace(no) / Retire(no) : Screen
    data object Hall/Pickup/Tool/Maintenance/Safety/Schedule/Performance/History/
        Messages/Notice/Help/Service/Settings/Feedback : Screen
}
```

- 进详情：`nav.push(Screen.Scan(no))`；返回：`nav.pop()`；切根页：`nav.switchTab(Screen.Home)`
- 工单号跨页传递：详情类页面都带 `no: String` 参数

## API 层（WorkerApi.kt 已定义，禁止修改）

```kotlin
AuthApi { smsCode(phone) / login(phone,mode,credential) / logout() }
HomeApi.get()
TicketApi { list(status?) / history(period?) / detail(no) / accept(no) / checkin(no,lat,lng) /
    navi(no) / transfer(no,reason,targetId,remark) / reschedule(no,date,slot,reason,remark) /
    rollback(no) / retry(no) / complaint(no,category,content) / repairReport(no,result,remark) }
HallApi { list() / grab(no) }
ScanApi { bind(no,epc,offline) / abnormal(no,payload) / photos(no) / uploadPhoto(no,scene) /
    report(no) / submitReport(no,remark) / activation(no) / activate(no) / sign(no,signatureData) /
    charge(no) / submitCharge(no,amount,payMethod) }
AssetApi { dismantleScan(no,epc) / replace(no) / submitReplace(no,oldEpc,newEpc) / returnAsset(epc) /
    materials() / materialOut(id) / tools() / borrowTool(id) / giveBackTool(id) /
    maintenance() / measure(no) / resources(no) }
ProfileApi { get() / performance(period?) / schedule(month?) / clock(IN|OUT) / settings() /
    saveSettings(s) / feedbacks() }
MiscApi { messages() / readAll() / clear() / notices() / faq(keyword?) / serviceMessages() /
    sendServiceMessage(content) / safetyCheck(workType,checklist) }
```

- 全部 `suspend` 函数，返回 `org.json.JSONObject`（`getArray` 返回 `JSONArray`）
- 失败抛 `ApiException`；页面内捕获并提示，不白屏

## 通用组件（Widgets.kt，禁止修改）

- `TopBar(title, onBack, action, onAction)` 渐变顶栏
- `Card(modifier, content)` 白底圆角卡
- `SectionTitle(title, more, onMore)`
- `Cell(title, desc, onClick, right)` 列表行
- `StatusTag(label, status, color)` 状态标签（status 映射色）
- `Notice(text, red)` 提示块
- `StatCard(title, date, values: List<Triple<String,String,Color>>)` 三列统计
- `KvRow(k, v)` 键值行
- `FieldLabel(text)` 表单标签；`OutlinedTextField` 用 material3 自带
- `PrimaryButton(text, modifier, enabled, onClick)` 绿色主按钮
- `Loading()` / `ErrorRetry(message, onRetry)` / `Empty(text)`
- `StatusLine(text, online)` 蓝点状态行
- `Load` 密封类 + `loadOnce(vararg keys, loader)` 异步加载（`Load.Loading/Ok/Fail`）
- `TicketCell(item, onClick, rightExtra)` 工单列表行

## 主题色（theme/Color.kt）

`Primary(0xFF1677FF) Primary2 Bg Panel Line Ink Muted Success Warn Err`
`TagColor(fg,bg)` + `TagGreen/Blue/Orange/Red/Gray/Cyan` + `tagColor(status)` 映射
页面背景用 `Bg`（F5F6F8）。

## 编码规范

- Kotlin，Compose material3，不用 View 体系
- 异步一律 `loadOnce` / `rememberCoroutineScope().launch`；UI 状态用 `var x by remember { mutableStateOf(...) }`
- 操作按钮（领取/签到/提交/上报）成功后用返回 `message` 提示（`Toast` 或页面内 `Notice`），再 `nav.pop()` 或刷新
- 不要 emoji 图标；中文文案与 HTML 草稿保持一致
- 草稿里的 `alert/confirm` → `Toast`/对话框；`location.href` → `nav.push/pop/switchTab`
- 编译验证：`cd mobile/worker/android && JAVA_HOME=/opt/homebrew/opt/openjdk@17 ./gradlew assembleDebug -q`

## 开发流程

1. 用 read 工具读草稿 `docs/worker/<page>.html` + `docs/worker/style.css`（视觉参考）+ 对应 API 方法
2. 用 read 工具读 `ui/Widgets.kt`、`ui/HomeScreen.kt`、`ui/OrdersScreen.kt` 作为风格参照
3. 写自己的 `<ScreenName>Screen.kt`，只 import 自己需要的
4. 编译验证自己的文件不破坏整体构建
5. 报告：文件路径 + 每个页面对应的 Screen 条目 + 需要主会话在 AppRoot 接线的地方
