# user Android 首发发版 checklist（D-1 签字项）

> 来源：2026-08-27 上线计划会议五、D-1（发版 checklist 签字）与 D-5 发布基建。
> 规则：**P0/P1 零放行，质量负责人独享停发权**；每一项有具体验证命令与预期结果，签字=逐项跑通。
> 积分/弱网/打磨等执行进度见 `meeting-minutes/2026-08-27-user-android-two-week-launch.md`。

## 签字项

| # | 项目 | 验证命令 / 预期 | 状态 |
|---|------|-----------------|------|
| 1 | 签名验证 | `scripts/user-android-release-sign.sh`：apksigner verify 通过 + release 指纹 ≠ debug 指纹；指纹见 adopted note 出包留档 | 本轮已跑通 |
| 2 | 版本号对齐 | `app/build.gradle.kts` versionCode/versionName；单调递增表见 adopted note（首发 2/0.1.0）；UpdateApi 自更新判定依赖同表 | 已定档 |
| 3 | cleartext 收敛 | Manifest 无 `usesCleartextTraffic`，`res/xml/network_security_config.xml` 仅白名单 102 内网/联调/模拟器回环；`aapt dump xmltree app-release.apk AndroidManifest.xml` 核对 | 已实现+物证（release 包：usesCleartextTraffic 0 处、networkSecurityConfig=@0x7f140002、无 ACCESS_BACKGROUND_LOCATION） |
| 4 | 权限实测 | 真机(Android 13+)冷启弹 POST_NOTIFICATIONS；拒绝/允许两条路径不崩溃；定位权限按场景触发（确认无 ACCESS_BACKGROUND_LOCATION） | 模拟器已实测，真机待测 |
| 4b | 脱敏验收 | 页面展示一律 `phoneMasked/idNoMasked`；明文手机号仅拨号盘 ACTION_DIAL（联系师傅设计意图），不落屏幕文案 | 审计通过 |
| 5 | UpdateApi 全链路 | `GET /client/latest` 出新版本 → App 启动弹升级提示 → 下载安装（后端版本元数据端点由后端 D-7 补齐；未就绪时回滚仅剩 adb 人工） | 后端待交付 |
| 6 | 多语言+实名回归 | 语言切换失败保留本地高亮；实名三态(未提交/审核中/驳回)+驳回重新提交入口走查 | 已核验 |
| 7 | 出包记录归档 | release 证书 SHA-256 指纹 + 版本 + 构建时间落 adopted note（本次:`1C:AC:B3:E1:03:CE:87:6E:9E:22:C9:FD:C8:D2:08:7C:ED:BF:45:58:A4:10:92:F8:EC:25:02:16:00:EE:F2:DA`） | 已归档 |
| 8 | 关键路径回归 | `connectedDebugAndroidTest` 全绿（当前 8/8）+ 6 条关键路径手测（登录真码→产品→下单→支付→订单/账单→实名） | 自动化部分已绿 |

## D10 稳定性基线（2026-08-27 实测）

- **冷启 <3s**：干净模拟器(swiftshader 软渲染)3 次 `am start -W -S` TotalTime = 1229/1196/1179ms，均值 ≈1.2s；首启含通知权限弹窗场景 2277ms——均 <3s 红线，真机更快。测量注意：权限弹窗会顶替 topResumedActivity 导致 `am start -W` 报 0，须先 `pm grant ... POST_NOTIFICATIONS` 再测。
- **无 ANR**：connected 套件整跑无崩溃/ANR；连测用 AVD `test_device`（无头 `-no-window -gpu swiftshader_indirect`）。
- 压测 P95<1.5s / 监控告警 5xx：后端侧（102 环境 D-3 压测，明远）。

## 回滚通道

- 服务端 versionCode 白名单渐进放量 + 服务端开关指向旧 APK（后端灰度基建）；客户端 UpdateApi 自更新为人工通道兜底。