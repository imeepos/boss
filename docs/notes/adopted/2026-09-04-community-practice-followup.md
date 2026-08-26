# 支付收口后续缺陷修复计划与社区实践对照

日期:2026-09-04

## 社区实践检索结论

1. **OpenAPI path 解析**:不要依赖行内 `$ref` 形态作为唯一识别条件;Path Item
   既可能在根 manifest 用 `$ref` 转发,也可能在子契约文件直接声明。实践参考:
   [OpenAPI Generator regex validation issue](https://github.com/OpenAPITools/openapi-generator/issues/20079)
   及 [swaggest/openapi-go path key issue](https://github.com/swaggest/openapi-go/issues/49)。
   本批采用最小改动:按 portal 扫根 manifest + 对应子目录的两空格 path key,不引入 YAML SDK。

2. **CDP 浏览器会话**:需要多步登录/跳转时复用同一 Chrome profile,否则 cookie/localStorage
   隔离。参考 [Puppeteer persistent session practice](https://stackoverflow.com/questions/57987585/puppeteer-how-to-store-a-session-including-cookies-page-state-local-storage)
   与 [Cloudflare CDP persistent browser docs](https://developers.cloudflare.com/browser-run/cdp/)。
   本批在既有零依赖 cdp-capture 上追加 `--user-data-dir`,默认临时 profile 行为不变。

3. **Webhook 验收**:真实环境测试必须先记录旧 URL,重启/变化后验证发现脚本、配置 API、
   provider endpoint 三方收敛,不能只验证本地字符串。参考 [Test real webhooks in CI](https://www.hookbase.app/blog/github-action-setup-tunnel)。
   本批 102 实测 cf-stripe-boss restart 产生新 URL,cron 脚本拉平 config,channel/test 识别
   provider endpoint 尚未在同一 cron 窗口完成后端 guard 收敛,后续按 P0 继续观察。

## 本轮落地

- `a89fd0d2`: contract-sync 解析根 manifest + 子契约 path,回归测试;
- `5b254549`: cdp-capture 持久 profile;
- `729749b7`: bossctl 路由重新生成;
- `ba1536d6`: portal 隔离扫描修正。

## 遗留

- Stripe endpoint 与新 URL 的 guard 周期收敛需继续在 102 观察/确认;
- 真实 demo user 登录后非空列表截图仍待补证;
- cdp persistent flow 已能跨调用保留 localStorage,toast 需配合 network/eval 证据而非截图猜。

## 放弃项

不引入 OpenAPI/Swagger 大型 SDK,不替换既有 CDP 工具,不新增 webhook provider SDK,
不修改业务支付状态机。