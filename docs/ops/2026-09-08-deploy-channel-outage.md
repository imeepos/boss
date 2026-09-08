# 运维发现:deploy-102 部署通道静默停摆(2026-09-08)

## 现象

- T14(合并提交 3d8e6f90)与 W7 收尾推送到 main 后,deploy-102 流水线未产生新镜像:
  102 本地 registry 中 boss/server 与 boss/admin-web 的 sha tag 停留在 58cf02e6 时代(约 4 小时前),
  5180 部署版 bundle 无新代码特征串,容器 boss-admin-web 启动时间早于推送时间。
- W7 会话曾推 retrigger 提交 cabf68b2 意图补跑,同样未生效。

## 根因链

1. gitea-runner 容器在推送窗口期已死,后自愈重启(docker logs 显示 re-register 成功);
2. 重启后拉取任务持续失败:rpc error: dial tcp: lookup postgres on 127.0.0.11:53: i/o timeout
   ——runner 容器内 DNS 解析 postgres 失败,疑似重启后未正确挂回 gitea 栈所在 compose 网络,
   或 gitea 栈 DNS 别名变更;
3. 期间宿主 sshd 间歇性 banner exchange 超时(高负载表现),HTTP 服务(3001/5180/28080)始终正常。

## 已采取措施

- 负责人会话(T14-3 收尾)改走 CI 等效手动部署:本地 cabf68b2 源码 rsync 至 102:/home/imeepos/boss-src,
  按 deploy-102.yml 同款命令构建 server+admin-web 双镜像、推 5000 registry、compose 拉起并自检。

## 待办(归基础设施会话)

1. 修复 gitea-runner 容器网络(核对 compose 网络挂载与 gitea 实例地址解析);
2. 观察一周:deploy-102 是否再现「推送无镜像」静默——建议 deploy-guard-alert.sh 增加镜像 tag 新鲜度巡检
   (对比 gitea/main HEAD 与 registry 最新 sha tag,滞后超阈值即告警);
3. runner-image-guard 只守护 job 镜像,不覆盖「runner 在线但拉不到任务」的半死态,考虑补探针。
