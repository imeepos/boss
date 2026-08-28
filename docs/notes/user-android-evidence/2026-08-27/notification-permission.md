# POST_NOTIFICATIONS 运行时权限实测（API 35 模拟器 emulator-5554）

- 日期：2026-08-27（执行推进）
- 设备：api35-notif AVD（Android 15 / API 35），补充佳宁「缺 13+ 设备」缺口
- 实现核查：MainActivity L46-50 `ActivityCompat.requestPermissions(..., 1001)` + Manifest 声明（API 33+ 运行时请求）

## 三态实测结果（uiautomator dump + dumpsys 断言）

| 路径 | 操作 | dumpsys 结果 | 结论 |
|---|---|---|---|
| 弹窗触发 | 启动 App（首次安装后） | 系统弹窗「Allow BOSS用户端 to send you notifications?」Allow/Don't allow | ✅ 弹窗正常出现（API 35 可触发） |
| 允许 | tap Allow (540,1315) | POST_NOTIFICATIONS: granted=true, flags=[USER_SET...] | ✅ 允许生效 |
| 拒绝 | pm clear → 重启 → tap Don't allow (540,1472) | granted=false, flags=[USER_SET...]；pidof 存活 | ✅ 拒绝后 App 不崩溃 |

## 结论

- POST_NOTIFICATIONS 非代码缺口：声明 + 运行时请求 + 三态处理全部正确。
- 佳宁此前「Android 11 无法实测」的缺口已由 API 35 模拟器补上，证据链闭合。
