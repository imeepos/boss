# API 在线文档:契约运行时聚合 + Swagger UI 入驻后台(api-docs)裁定

日期:2026-09-06。需求:把仓库内 OpenAPI 契约变成管理员可用的在线文档,集成进后台管理。

## 裁定(Why)

1. **单一事实源即仓库契约**:`api/openapi/**.yaml` 经 `go:embed` 打进 server 二进制,
   `internal/pkg/openapidoc` 运行时聚合(外部 $ref 原位展开、schema 上提去重、
   各文件本地 components 并根),`GET /api/admin/v1/docs/openapi?portal=admin|user|worker|open`
   下发自包含 OpenAPI 3 JSON。不再另建文档站数据源——契约改了文档自动变,
   消灭"文档与实现两处维护"漂移。
2. **渲染引擎选 Swagger UI(非 Redoc/Scalar)**:Try-it-out 让管理员在页内真实
   调接口;`requestInterceptor` 把契约相对 servers 前缀改写到服务端配置基址并注入
   当前 admin JWT——权限仍由后端逐接口 RBAC 拦截,文档页不放大攻击面。
3. **门禁对齐页面**:`menu:apidocs`(迁移 000166,授 sysadmin),契约 Docs 域
   同权限码;非 sysadmin 角色按需在菜单权限页勾选。
4. **惰性加载**:swagger-ui-react 独立 chunk(约 1.2MB),仅进入该页时加载,
   主包不受影响;暗色主题用 --shell 令牌覆盖主表面(apidocs.css),未新增令牌。
5. **CI/部署注意**(本次实坑):
   - swagger-ui-react 传递依赖(@scarf/scarf、tree-sitter*、core-js-pure)必须
     在 `web/admin/pnpm-workspace.yaml` allowBuilds 显式 false,pnpm 11 否则
     frozen-lockfile 硬失败。
   - deploy-102 工作流按"最后部署 sha"分类变更,docs-only 提交会跳过部署;
     并发取消可能截断 image push(本次 `latest` 未更新,sha 标签已推)。部署后
     验收以 `scripts/ops/apidocs-acceptance.sh` 为准,发现 404/403 立即核对
     102 容器镜像 id 与 registry 是否一致。

## 放弃了什么

- Redoc/Scalar:阅读体验好但不带 Try-it-out(Scalar 体积与 API 稳定性风险)。
- 独立文档站(Mintlify/Docusaurus):引入第二数据源,违背单一事实源目标。
- 无鉴权公开文档端点:契约暴露面收敛在 admin JWT + menu:apidocs 内。

## 附带发现(接入真实解析器暴露的契约存量债,已同轮修复)

regex 消费时代遗留:重复键、追加在 `components:` 之后的错位 path 块、悬空
$ref(/olt-devices、/backup/*、schemas/Item 等)、admin 根缺 securitySchemes——
详见 commit ca216571 fix(contract) 正文。教训沉淀于 self-evolving techniques.md:
接手只被正则消费过的 YAML 资产,先做全树解析 + 悬空引用扫描。
