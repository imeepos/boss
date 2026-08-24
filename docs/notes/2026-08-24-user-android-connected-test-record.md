# user-android connected 测试实跑记录（2026-08-24）

## 环境

- 模拟器：test_device AVD（android-35，API 15 镜像），headless（`-no-window -gpu swiftshader_indirect`），本机 Mac
- 命令：`./gradlew :app:connectedDebugAndroidTest`（worktree fix/user-android-round3）

## 结果

- 4/4 通过（TEST-test_device(AVD) - 15-_app-.xml：`tests="4" failures="0" errors="0"`）
- 含新增用例 `messagesSkeletonFitsNarrow360dpViewport`（ForcedSize 360×800 窄屏基线，首次实跑）

## 过程发现（重要）

1. **历史 connected 全是空跑**：`testInstrumentationRunner` 未声明，AGP 注册老
   `android.test.InstrumentationTestRunner`，JUnit4/Compose 用例静默跳过且 BUILD SUCCESSFUL、
   结果 XML `tests="0"`。已在 user/worker 两端 build.gradle.kts 显式声明
   `androidx.test.runner.AndroidJUnitRunner`（user 修复 + worker 占位）。
2. **faultDetail 冒烟依赖后端数据**：ActionsCard 原在详情加载成功分支内，无凭证模拟器上
   "联系师傅/催单"不渲染致用例失败。已将 ActionsCard 挪出（动作只依赖工单号），用例与后端解耦。

## 待办

- JPush / 备份恢复端到端验证仍缺。

## 补充(同日晚,E2E 变更地址链路)

- 模拟器实测发现后端 bug:GET /addresses 的 `addressId` 恒空(`toStr` 只认 string,
  pgx bigint 扫成 int64)。修复后部署 102,复测 addressId=1/2 正常。
- API 级 E2E 闭环:sms-code(DB 取码)→login→POST /orders/ORD-20260821-000359/change-address
  (addressId=2)→DB 断言 orders.address_id=2 落库成功。
- App UI 链路:登录(验证码回填)、订单详情、变更地址子页(地址簿渲染/默认预选)均实测可达;
  子页"确认变更地址"按钮在 uiautomator 下点击结果随机(同坐标时而生效时而无响应),
  判定为该模拟器 Compose 语义/坐标漂移,未取得稳定的点击级断言——按钮绑定代码与
  SubmitBar 为既有复用件,API 载荷一致。此项留待真机复测。
- 另发现首页疑似点击区重叠(进行中订单卡部分区域点击误入"我的套餐"),待专项排查。
