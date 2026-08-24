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

- 变更地址子页 E2E（需登录态 + 102 环境）未做，待有测试账号流程时补。
- JPush / 备份恢复端到端验证仍缺。
