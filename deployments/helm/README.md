# Helm Chart(阶段12 W12 落地)

Chart 位于 `boss/`:`helm lint` 通过,渲染产物 5 Deployment(server/collector/provisioner/gis/report)+ Service(HTTP/gRPC 双口)+ HPA。

```sh
helm upgrade --install boss ./boss \
  --set image.repository=<registry>/boss-server \
  --set image.tag=<git-sha> \
  --set env.BOSS_DATABASE_DSN="<生产 DSN>"
```

- 探针:server /healthz(readiness+liveness);/metrics 供 Prometheus(deployments/observability/prometheus.yml 已配 boss-server 抓取)。
- 组件开关:components.collector/provisioner/gis/report 按灰度需要开关。
- 依赖:bitnami/postgresql、bitnami/kafka、bitnami/redis-cluster、temporal(生产建议托管服务)。
- 回滚:`helm rollback`;迁移层见 docs/plan/launch-checklist.md(000034 down→up 已演练)。
