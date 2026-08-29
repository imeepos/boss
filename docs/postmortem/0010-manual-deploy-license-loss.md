# 0010 手动部署致授权证书失效(postmortem)

- 日期:2026-08-29
- 级别:P2(授权失效,业务功能受限约 40 分钟;数据无损)
- 现场会话:柜面收款功能部署轮(feat/counter-close-loop、feat/counter-dropdown 合并后)

## 时间线

1. 会话执行柜面收款部署时**未走 CI**,在 102 `~/boss/deployments` 手工
   `docker build`(102-remote builder)+ `docker compose up`。
2. 该目录的 compose 项目名为 `deployments`,与 CI 部署项目 `boss-app` 不同——
   docker 卷按项目名隔离:容器重建后挂到了新建空卷
   `deployments_boss_license_data`,`/var/lib/boss/license.json` 消失。
3. `/license/status` 返回 `activated:false`,系统授权页显示"未激活·业务功能受限"。

## 根因

- 直接原因:换卷部署丢证书文件(卷按 compose 项目名隔离)。
- 制度原因:**违反 docs/deploy/oncall-102.md 既有部署契约**——"部署 =
  gitea CI(Build-Deploy-to-ECS),push main 即自动部署+自动迁移"。
  手动 docker build 未注入 `BOSS_LICENSE_PUBLIC_KEY_HEX`(镜像沦为开发态),
  且手工替换容器改变挂载卷,两处都越过了 CI 的标准状态。

## 恢复动作(已验证)

1. 定位 CI 卷 `boss-app_boss_license_data`(license.json 完好,Aug 28 17:21);
2. 复制 license.json 到现挂载卷,宿主侧 `chown 1000:1000`(CI uid 对齐);
3. 重启 boss-server → `/license/status` = `activated:true`,
   licenseId ac-3f41ec806850247d,有效期至 2026-09-27,指纹匹配;
4. 业务复验:柜员收款(id=63)、网点清单端点均正常。

## 教训(硬规则)

- **102(及任何环境)部署只允许 push main 触发 CI;严禁手动 docker
  build/compose up**。手动部署=绕过证书注入/卷/迁移三条标准链路。
- 部署类操作前必读 docs/deploy/oncall-102.md(本次会话未先读)。
- 镜像构建工具链:`make build-server` 对空公钥 fail-fast 是正确的,
  docker build 路径缺同等拦截,后续可加 BuildKit check(候选优化)。

## 待办

- [ ] gitea Build-Deploy-to-ECS 触发一次全量部署,消除 deployments 项目
      遗留容器(或运维手工 `docker rm` deployments 项目容器+卷)。
- [x] server.Dockerfile 增加 ARG 非空校验(构建期 fail-fast):空公钥且未显式 ALLOW_DEV_LICENSE=1 → 构建失败;双向实测(开发态可过/无公钥被拦)2026-08-29。
