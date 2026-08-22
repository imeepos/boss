# 0008 AAA 管理二期部署边界

## 现象

二期 AAA 管理分页、筛选、数据范围和总览聚合已合入 main。102 在本轮验证时 `/aaa/summary` 已返回新总览数据，但 `/lo-accounts?pageSize=2` 仍返回旧 `{items}` 全量形态，没有 `total/page/pageSize`。

## 原因

102 `boss-server` 镜像创建时间为本轮合并前，已部署的是上一版 AAA 总览代码，不是本轮分页收敛代码。HTTP 200 和 summary 正常不能证明所有新接口已部署。

## 处置

不在远端手工改业务容器，保留代码和迁移在 gitea/main，等待 CI/CD 构建并部署最新镜像。完成后必须重新验证：

- `/lo-accounts?page=1&pageSize=2` 返回两条且含 `total/page/pageSize`
- `/cdrs?page=1&pageSize=2` 返回分页 envelope
- `/auth-logs?page=1&pageSize=2` 返回分页 envelope
- `/aaa/summary` 返回数据库聚合结果
- Admin 页面请求使用分页参数且 console 无错误
