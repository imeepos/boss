# 移动端 Android 初始化:Kotlin + Compose 双 app 独立工程(2026-08-19)

> Amended 2026-08-19: 当前目录先保持 `mobile/user` 与 `mobile/worker` 各自独立，不预建 `mobile/shared`；未来确有共享能力时另立决策。

## 决策
mobile/user/android 与 mobile/worker/android 各自初始化为完全独立的 Gradle 工程,
不建 monorepo 复合构建(composite build),不共享 module:

- 技术栈:Kotlin 2.1.21 + Jetpack Compose(BOM 2026.06.00, material3)+ AGP 8.13.1,
  compileSdk/targetSdk 36,minSdk 26,JDK 17,Gradle wrapper 8.14.3。
- 包名/applicationId:com.ymm.boss.user / com.ymm.boss.worker,与 go module
  github.com/ymm-001/boss 的域命名对齐(worker=师傅端,user=用户端)。
- 版本目录统一走 gradle/libs.versions.toml,两个工程模板完全同构,仅包名/label/工程名不同。

## 为什么
- 契约边界(docs/plan/directory-spec.md):师傅端与用户端不互相放实现代码、不跨 app
  共享页面实现;独立工程在构建层面强制该边界,共享能力放 mobile/shared(待建)。
- 版本选择依据本机 gradle 缓存(~/.gradle/caches 与 wrapper/dists)中已存在的完整组合,
  首次构建即成功,避免为追新版本拉全量依赖。

## 放弃了什么
- 放弃复合构建/单 settings 多 app:换取两个 app 可独立构建、独立 revert;
  若后续抽公共 UI/协议层,以 mobile/shared 源码集或独立 AAR 引入,届时本文 Amended。
- 放弃 minSdk 24 以下:Compose material3 1.4 实际要求 21+,取 26 覆盖目标市场即可。
- 放弃 Flavors 多环境:后端地址等环境配置留待网络层引入时再定,不在骨架期拍板。

## 环境事实
- 本机无系统 JDK,Android 构建用 brew openjdk@17(/opt/homebrew/opt/openjdk@17)。
- services.gradle.org 网络不可达(wrapper 校验会失败),但 gradle-8.14.3 发行版
  已在 ~/.gradle/wrapper/dists 缓存,wrapper 直接可用;依赖走 google()/mavenCentral 在线拉取。
