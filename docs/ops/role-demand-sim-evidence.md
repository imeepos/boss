# 全角色需求模拟实跑证据(S12/T22,2026-09-05)

- 环境: 102 对接环境 http://192.168.0.102:28080(bossctl 免登录 key,真实读写)
- 套件: scripts/ops/role-demand-sim.sh(8 角色 17 场景,自带验收锁与 acc_ 造数清理 + 孤儿巡检门禁)
- 实跑留档: docs/ops/evidence/role-demand-sim-2026-09-05.log(47 行,PASS 行 17,FAIL 行 0)
- 实跑结果: PASS=17 FAIL=0,末行 ROLESIM-ALL-PASS,退出码 0

## 一、17 场景结果表(8 角色 x 17 场景)

| # | 角色 | 场景 | 结果 | 判据(一句话) |
|---|------|------|------|--------------|
| 1 | customer | 客户自助下单 | PASS | 两次下单均成功返回订单号,订单可按单号检索 |
| 2 | customer | 取消未受理订单 | PASS | 取消后状态 CANCELLED,重复取消被拒 |
| 3 | customer | 施工段改地址被拒 | PASS | DONE 单 change-address 被拒(非法转移),address_id 不变 |
| 4 | dispatch | 资源核查预占与派单(段2/3/8) | PASS | check-resource/reserve 均 200 且 ports 表出现 RESERVED 记录,工单派发成功 |
| 5 | dispatch | 资源不足失败可观测 | PASS | 空资源地址 check-resource 被拒并返回明确错误,失败路径可观测 |
| 6 | cashier | 合同收费(段4) | PASS | charge 200 且订单 stage 推进到 >=4 |
| 7 | cashier | 重复收费不落账与前置不符被拒 | PASS | 前置不符的 charge 被拒;重复收费状态无漂移,不重复落账 |
| 8 | worker | 未扫码先激活被拒(时序异常) | PASS | 未扫码直接激活被拒,时序闸门生效 |
| 9 | worker | 接单扫码激活签收 | PASS | 接单-扫码-激活-签收全链 200(附 1 条 tl1sim 未完结 INFO 观测,见结论) |
| 10 | kefu | 订单查询与投诉受理办结 | PASS | 客服 key 可查订单并完成投诉受理-办结闭环 |
| 11 | kefu | 越权下单被拒(账号主体不可冒客户) | PASS | 客服主体直接下单被拒 |
| 12 | noc | 端口查询与置备 | PASS | 置备成功且端口清单含 P-RLS 标记端口 |
| 13 | noc | 跨域调财务接口被拒 | PASS | noc key 调财务接口被拒(403),域边界生效 |
| 14 | reviewer | 注册审批通过与驳回 | PASS | 客户注册 approve 成功,师傅注册 reject 成功 |
| 15 | reviewer | 驳回后重新提交再审闭环 | PASS | 重新提交后再审批通过,师傅状态 APPROVED |
| 16 | admin | 受权建号与签发APIKey | PASS | 建号成功,签发新 key 且新 key /auth/me 鉴权通过 |
| 17 | admin | 客户师傅key调admin被拒(三表边界) | PASS | customer/worker key 调 admin GET /orders 均被拒 |

## 二、角色核心诉求与场景映射

| 角色 | 核心诉求 | 对应场景(#) |
|------|----------|------------|
| customer | 自助下单/取消、安装信息不可被施工段静默篡改 | 1,2,3 |
| dispatch | 资源核查-预占-派单顺畅,资源不足时有明确失败信号 | 4,5 |
| cashier | 收费准确推进订单,重复/前置不符收费不落账 | 6,7 |
| worker | 按扫码时序激活签收,时序异常被拦截 | 8,9 |
| kefu | 查订单、处理投诉,但不能代替客户下单 | 10,11 |
| noc | 端口/资源置备履职,但不可越权碰财务 | 12,13 |
| reviewer | 注册审批通过与驳回、驳回后重审闭环 | 14,15 |
| admin | 受权建号与 APIKey 签发,三表(admin/user/worker)边界不可穿透 | 16,17 |

## 三、缺陷清单(T21 实跑发现,已修复,本轮回归)

### FIX-1 0b68e7f7 fix(rbac): 补授 resource_admin 的 menu:provision 置备权限

- 现象: T21 实跑中 noc_chen(resource_admin)调 POST /provision/resources 返回 403 no permission: menu:provision,无法履行网维端口/资源置备职责;而账号台账标注 noc 负责域即含 /provision/*。
- 根因: /provision/* 置备路由(资源/端口/渠道/标签/资产)统一挂 menu:provision 门禁,000003 种子只授了 sysadmin,resource_admin 仅得 menu:resource 等查询位,属种子遗漏而非设计。
- 修复: 迁移 000183 补授(对齐 000080 menu:odn 先例),down 迁移对称回收,可独立 revert。
- 回归结论: 本轮场景 12(noc 端口查询与置备)PASS,置备链路全通,无回归。

### FIX-2 3b8be8f1 fix(order): change-address 仅限 PENDING 单,施工段静默改地址判非法转移

- 现象: T21 实跑中订单 DONE 后 change-address 仍返回 200,orders.address_id 被静默改写;派单工单站点坐标在派单时刻物化、端口已锁定安装地址,放行会造成工单/端口与订单地址漂移,半径闸门与四码口径失真。
- 根因: PGStore 与 MemoryService 的 ChangeAddress 均为无状态守卫的直写 UPDATE,缺状态闸门。
- 修复: 补 PENDING-only 闸门,非 PENDING 返回 ErrIllegalTransition(经 httpx 既有映射落 42200,失败可观测);内存实现同口径并附回归测试(覆盖 PENDING 放行与 RESERVED/INSTALLING/DONE/CANCELLED 四态拒绝)。
- 回归结论: 本轮场景 3(施工段改地址被拒)PASS,DONE 单改址被拒、地址不被改写,无回归。

## 四、结论

8 个角色(customer/worker/kefu/dispatch/cashier/noc/reviewer/admin)在本轮 102 真实环境实跑中均可正常处理各自职责范围内的各类需求数据:正向链路(下单-取消、核查-预占-派单、收费、扫码激活签收、投诉办结、置备、审批、建号发 key)17 项全部 PASS,负向边界(越权、非法转移、时序异常、重复收费、跨域)也全部按预期拦截,PASS=17 FAIL=0。

各角色不可处理项(均为设计内的权限/状态边界,拦截即正确,非缺陷):

1. customer: 施工段(非 PENDING)改安装地址不可处理,被 422 非法转移拦截(场景 3)。
2. kefu: 不可冒客户主体下单(场景 11)。
3. noc: 不可跨域调财务接口(场景 13)。
4. worker: 不可未扫码先激活(场景 8)。
5. cashier: 前置不符不可收费,重复收费不落账(场景 7)。
6. customer/worker key: 不可穿透三表边界调 admin 接口(场景 17)。
7. 其他 admin/受权能力: 无不可处理项。

遗留观测(不阻塞): 场景 9 附一条 [worker] INFO 下发任务未完结(tl1sim 环境暴露): 1 项,为 tl1sim 模拟环境下发任务收尾时点的观测信号,链路本身全通;后续若在真实施工环境复跑,该信号应自然消失。
