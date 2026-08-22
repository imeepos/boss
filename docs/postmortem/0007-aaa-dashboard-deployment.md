# 0007 AAA 管理总览部署核验

## 现象

AAA 管理总览页面和 `/aaa/summary` 已在代码中完成并合入 `main`，但 102 环境仍运行旧镜像。访问真实 102 `/api/admin/v1/aaa/summary` 返回 HTTP 404；既有 `/lo-accounts` 仍正常返回，说明不是认证失败或 AAA 数据缺失。

## 原因

本轮完成的是代码提交、分支推送和本地门禁，未触发/等待 102 的应用镜像构建部署。102 `boss-server` 镜像创建时间早于本轮合并提交。

## 处置

未直接在 102 上手工替换业务镜像，避免绕过部署流水线。代码已提交至 `gitea/main`，待 CI/CD 构建 `boss/server:latest` 并滚动部署后，再复测 `/aaa/summary` 与页面真实请求。

## 后续

部署完成后必须使用 admin JWT 验证：

- `GET /api/admin/v1/aaa/summary` 返回 `code=0` 且含 `summary`
- `GET /api/admin/v1/lo-accounts` 仍返回真实 `billingMode`
- `GET /api/admin/v1/cdrs` 与 `/auth-logs` 正常
- 浏览器访问 `/aaa/dashboard`，断言标题、真实统计卡和无 console/网络错误
