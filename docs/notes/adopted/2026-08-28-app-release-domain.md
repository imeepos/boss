# 客户端版本发布域(apprelease, 000137)设计裁定

日期:2026-08-28。需求:官网首页双端下载入口、后台版本管理(上传/灰度/白名单)、
两端在线更新弹框(可忽略)、个人中心检查更新。

## 裁定(Why)

1. **升级判定双门槛 + 确定性灰度分桶**(社区通行,mPaas/juejin 更新接口设计同款):
   - `versionCode` 单调解耦展示 semver 与升级判定;
   - `minSupportedCode` 以下的客户端强制更新,其余弹框可忽略(需求明确"不强制");
   - 灰度命中 = `sha256(deviceId+releaseId)%100 < rolloutPercent` 或白名单命中,
     确定性哈希保证同一设备灰度期间稳定(不闪变),未命中回落最近 PUBLISHED。
2. **单一 client_releases 表**,app 字段区分 user/worker 两端,不建两张表:
   两端发版语义完全同构,双表只复制迁移与 handler。
3. **APK 对象入 MinIO,复用 attachment 的 ObjectStorage 抽象**,不落本地磁盘:
   102 已有 MinIO,本地盘会在多实例/回滚时丢包。上传上限单独放大到 256MB
   (attachment 32MB 不够 APK;不复用其 multipart handler 的原因)。
4. **免登录检查端点挂在各自端的 pub 组**(`/api/worker/v1/client/latest`,
   `/api/user/v1/client/latest`),不放 open(HMAC 面向集成方)也不放 admin 匿名组:
   版本检查发生在启动/未登录时,端前缀即客户端自己的契约面。
5. **官网下载入口走 admin 匿名组** `/client-releases/latest?app=`,沿用 site/posts 先例。

## 放弃了什么

- 不做差分包/热修复(国内常见方案):两端均为自建分发出海 App,全量 APK + sha256
  校验足够,差分复杂度先不背。
- 不接 Google Play In-App Update:菲律宾市场走自建分发(现有 CI 已出 APK 到 102)。
- 不做多渠道(channel)字段:当前只有官网+App 内两条下载路径,引入渠道统计是过早设计。
- 状态机不做 OFFLINE:回滚用 ROLLED_BACK,语义是"曾全量、现撤回",与下线区分。
- 强制更新虽然需求说不强制,仍保留 force 字段+minSupportedCode:留给未来的
  安全修版本,默认 false 即满足"可忽略"。

## 迁移

000137_client_releases:表 + menu:client-release 权限种子(授予 sysadmin,
沿 000135 cms_menu 先例:权限码迁移随菜单走)。
