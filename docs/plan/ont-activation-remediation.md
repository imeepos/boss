# 光猫授权与解锁链路修复计划

## 依据

- 订单 12 环节契约：扫码绑定为环节 9，激活为环节 10，激活回调为环节 11，见 `docs/contract/terms.md`。
- AAA、配置下发、四码合一分别归属 `aaa`、`provision`、`quadlink` 域，跨域只通过服务口/事件编排，见 `docs/contract/domain-map.md`。
- RADIUS 认证遵循 [RFC 2865](https://datatracker.ietf.org/doc/rfc2865/)，队列领取参考 PostgreSQL `FOR UPDATE SKIP LOCKED` 实践：[Netdata queue workflows](https://www.netdata.cloud/academy/update-skip-locked/)。

## 分阶段执行

1. **安全门禁**：师傅端 report/activate 必须校验工单归属；补回归测试并独立提交。
2. **状态真实性**：激活接口不能直接以订单 stage 代表网元成功；增加真实任务/授权结果读取，移除硬编码成功字段。
3. **账号与预配置**：环节 6 幂等创建 LOID 账号；环节 7 幂等创建 provision task，绑定订单、LOID、模板和资源；补 PG 集成测试。
4. **统一激活编排**：worker/admin 复用统一服务；真实 provision 成功、AAA 授权确认后才推进环节 10-12；失败可重试并留痕。
5. **执行器可靠性**：provision 使用原子 claim/lease；模板参数真实渲染；统一重试计数。
6. **AAA 完整性**：认证结果写 `auth_logs`，区分 ACTIVE/SUSPENDED/CLOSED，修复 Accounting 错误吞没；PAP/CHAP 仅在明确 NAS 契约后接入。
7. **门禁与合并**：每阶段测试通过即 commit；最终执行 Go 测试、vet、契约检查和真实 102 冒烟，推送 gitea，主树 ff-only 合并，随后删除 worktree 和短命分支。

## 当前阶段边界

本分支先完成安全门禁。没有厂商 OLT 私有协议和真实设备测试契约时，不伪造“解锁成功”；后续实现必须以 Executor 的真实返回和可查询结果驱动订单状态。
