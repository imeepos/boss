# U2000 北向 TL1 接口接入方案（2026-09-02，仅方案未动代码）

## 0. 输入资料

| 资料 | 内容 | 定位 |
|:-----|:-----|:-----|
| `docs/books/iManager U2000 V200R014C60 北向TL1接口 用户指南 (电信企业规范) 10.pdf` | 1020 页，华为 U2000 北向 TL1 权威协议文档 | 协议/命令/响应格式的唯一事实源 |
| `docs/books/POWC_TL1指令.xlsx` | 现网真实开通动作捕获，6 条 TL1 指令 | 业务语义与参数取值的实证 |

## 1. 一句话结论

TL1 接入 = 为 provision 域**新增一个 U2000 协议执行器**（与现有 TelnetExecutor 并列，由配置切换），
协议本身简单（TCP 文本行协议）；真正的工作量在**参数解析**（BOSS 数据 → OLTID/PONID/ONUID/VLAN）
和**幂等/对账语义**，现有数据模型缺 TL1 定位数据，需一次小迁移。

## 2. TL1 协议要点（PDF §3/§10/§11 提炼）

- **传输**：U2000 为服务端，BOSS（OSS 角色）主动连 TCP。业务开通/存量查询/诊断 = **13027**，告警 = **13028**。
  连接方式 TCP/SSH/SSL 可配（缺省 TCP，文档建议 SSL）。全系统最大 30 连接，开通类建议 15~20 个——
  BOSS 用**单连接长会话**即可。
- **会话生命周期**：连接建立后必须先 `LOGIN:::CTAG::UN=xxx,PWD=xxx;` 鉴权（UN≤20 字符、PWD≤16 字符，
  是 U2000 的用户体系）；**10 分钟无通信自动断连** → 需 `SHAKEHAND` 定时保活；退出用 `LOGOUT`。
- **命令格式**：`verb[-mod[-mod]]:[target]:accessID:ctag:[payload];`
  - target 恒为空（写空冒号占位）；accessID 定位对象（如 `OLTID=xx,PONID=NA-0-7-5`）；
  - **ctag** 是请求-响应关联标签，响应原样回带——BOSS 必须用连接级自增唯一 ctag 配对，
    不能学 xlsx 用 `ADDONT`/`Internet` 这类语义标签（并发下撞车）。
  - payload 为 `k=v` 逗号分隔参数块。
- **响应格式**：头行（`HW_<网管IP> <日期时间>`）→ `M  <ctag> <完成码>` → `EN=<码>  ENDESC=<描述>` → 终止符。
  - 完成码：`COMPLD` 成功 / `DENY` 失败 / `DELAY` 延迟执行（继续等同 ctag 最终响应）/ `PRTL` 部分执行 / `RTRV` 测试中。
  - `EN=0` 即成功；**ENDESC 文案随版本会变，禁止作为判定键**，只入日志。
  - 查询类响应为表格（attrib 行 + value 行，TAB 分隔）；大数据分块（`blktag/blkcount/blktotal`），
    终止符 `>` 表示还有后续块，`;` 表示结束。
- **性能**：配置类命令 10 条/秒；响应 2 分钟内返回（测试类除外）。
- **License**：TL1 业务发放按 ONU/端口计（控制项 LNSDNELCR86 等）——接入前向局方确认已购。
- **参数字符集**：OCTET STRING 参数仅允许字母、数字和 `()_-+/\.空格`（xlsx 密码含 `@` 属口令域不受此限）。
  我们生成的 DESC 等值必须白名单化（如 `PRV-ORD-20260902-0001`）。

## 3. POWC xlsx 六条指令 → 业务语义

| # | 业务动作 | 指令（关键参数） | BOSS 对应场景 |
|:-:|:---------|:-----------------|:---------------|
| 1 | 登录 | `LOGIN:::POWC::UN=BOSSsystem,PWD=Boss@123;` | 会话建立（凭据入密钥管理，不入代码库） |
| 2 | 新增 ONT | `ADD-ONU::OLTID=172.16.16.2,PONID=NA-0-7-5:ADDONT::AUTHTYPE=MAC,ONUID=ZKXXC3102EE8,ONUNO=0,DESC=Test001,ONUTYPE=Internet;` | 环节7 预下发：SN 预注册（ZKXXC3102EE8 为 12 位 GPON SN；`ONUTYPE` 填 **模板集名称**） |
| 3 | Internet 业务流 | `ADD-PONVLAN:...:Internet::SVLAN=1113,CVLAN=1,UV=100,SCOS=0,CCOS=0;` | 环节7 上网业务（双层 VLAN） |
| 4 | TR069 通道 | `ADD-PONVLAN:...:TR069::CVLAN=1000,UV=1000,SCOS=6,CCOS=6;` | 环节7 管理 VLAN（单层，无 SVLAN） |
| 5 | 更改 SN | `CFG-ONU:...:Modify::AUTHTYPE=MAC,AUTHINFO=<新SN>;` | 换机/移机（P2） |
| 6 | 销户 | `DEL-ONU:...:DELONT::ONUIDTYPE=ONU_NUMBER,ONUID=0;` | 退订销户，连带删业务流（P2） |

补充可用的查询/验证类命令（PDF §13/§15）：`LST-ONU`（存在性）、`LST-ONUSTATE`（在线状态）、
`LST-PONVLAN`（业务流核对）、`LST-UNREGONU`（未注册 ONU 发现，可反哺装机辅助）。

## 4. 现状与差距

**已有可复用**（`internal/domain/provision/`）：

- Task/Template/Log 模型 + Daemon（`FOR UPDATE SKIP LOCKED` 领取、FAILED 重试计数、OnDone 通知）；
- `Executor` 口 = `Exec(ctx, Task) error`；`provision_templates.content` JSONB 已存在（000127）可承载 TL1 参数；
- 环节6 已建 `lo_accounts.loid`（LOID 认证可直接映射）；环节10/11 目前只推进状态机，无真实网元动作。

**四个缺口**（本方案要补）：

1. TL1 客户端不存在（会话/编解码/ctag 配对/保活/重连）；
2. 参数解析层不存在：Task（仅 TaskNo/OrderID/StageEvent/LoAccountID/TemplateID）→ OLTID/PONID/ONUID/ONUNO/VLAN 无取数路径；
3. 数据模型缺口：`resources`(OLT 行) 无 U2000 侧 OLTID 与网管端点；`ports` 无 PONID（框-槽-端口）定位；
   ONU SN 无处存（assets 无 sn 列）；ONUNO 无分配器；U2000 地址+凭据无配置位；
4. 幂等与对账：ADD-ONU 重跑会重复报错、DELAY 语义、失败重试需与网元真实状态对账（LST-ONU 预检）。

## 5. 目标架构

新子包 `internal/domain/provision/tl1/`（归属 provision 域，不跨域；每文件守 300 行红线）：

| 文件 | 职责 |
|:-----|:-----|
| `codec.go` | 命令构建（参数白名单转义）+ 响应解析（完成码/EN/ENDESC/多块续读/查询表格 → []map） |
| `session.go` | 连接状态机：dial → LOGIN → ready → SHAKEHAND(默认 3min) → LOGOUT；per-conn 串行写读按 ctag 配对；断线退避重连，重连后丢弃陈旧缓冲；会话级 DENY 重登录一次 |
| `client.go` | 业务命令封装：Login/Logout/AddONU/AddPONVLAN/CfgONU/DelONU/LstONU/LstONUState/LstPONVLAN → 结构化结果 + 类型化错误（ErrAlreadyExists 等） |
| `executor.go` | 实现 `provision.Executor`：按 StageEvent 编排命令序列；`ParamResolver` 接口取数 |

- **ParamResolver 实现放 internal/app 装配层**（聚合 order/aaa/resource/provision 读取），避免域间反向依赖；
- **wiring**：`cmd/provisioner/main.go` 加 `BOSS_PROVISION_DRIVER=log|telnet|tl1` 开关（默认维持现状，灰度切换）；
- **配置**（`internal/pkg/config`）：`Provisioner.TL1{Addr(默认:13027), Protocol(tcp), User, Pass, KeepaliveInterval, CmdTimeout(默认120s), MaxQPS(默认8)}`。

## 6. 环节 → TL1 命令映射

| 环节/事件 | TL1 序列 | 幂等策略 |
|:----------|:---------|:---------|
| 7 preConfigOLT | `LST-ONU` 预检 → `ADD-ONU` → `ADD-PONVLAN`(Internet) → `ADD-PONVLAN`(TR069) | 预检已存在则跳过；任一步 DENY 整体 FAILED 留痕可重试 |
| 10 activateUser | `LST-ONUSTATE` 核实 ONU 在线 | 查询类天然幂等；未上线 FAILED（装维装好 ONT 后重试），不伪造成功 |
| 11 notifyActivation | `LST-PONVLAN` 核对业务流存在 → SUCCESS 落 activation_callbacks | 查询类 |
| 退订/销户 (P2) | `DEL-PONVLAN` ×N → `DEL-ONU` | 网元不存在视为成功 |
| 换机 (P2) | `CFG-ONU` AUTHTYPE+AUTHINFO=新标识 | — |

**关键裁定（实施时按决策记录制度写 dated note）**：

1. **环节7 即做 ADD-ONU 预注册**——对齐 POWC 现网做法与 PDF §17/§18 预配置模型；
   环节10 不重复下发配置，只做状态核实，维持"配置就绪才可派单"既有契约。
2. **认证方式默认 `AUTHTYPE=LOID`**，`lo_accounts.loid` 直接映射（BOSS 是账号权威，换机只需 CFG-ONU 改绑，
   不依赖 SN 台账）；`AUTHTYPE=MAC/SN` 作为可选路径，启用前提是补 SN 台账（xlsx 现网用 MAC 是因为没有 BOSS LOID 体系）。

## 7. 数据模型与契约改动（迁移草案，实施前按占号规则核查编号）

| 改动 | 内容 |
|:-----|:-----|
| 新表 `provision_nms` | 法人级 U2000 端点：`legal_entity_id, host, port, protocol, user, pass_cipher`(参照 config_secrets 加密模式)——一个法人一个 U2000 |
| `resources` 增列 | OLT 行加 `nms_oltid`（U2000 侧 OLT 标识，即 TL1 OLTID，可取管理 IP） |
| `ports` 增列 | `pon_frame/pon_slot/pon_port`（拼 `NA-<frame>-<slot>-<port>` 即 PONID）+ `onu_no`（ONUNO，DEL-ONU 后回收） |
| `assets` 增列 (P1) | `sn`（启用 MAC 认证路径的前提） |
| 契约文档 | `fields.md` §下发 增上述列；`domain-map.md` provision 域增 tl1 子包；terms.md 12 环节不变，仅环节7/10 执行语义注记 |

模板 content 约定（JSONB，模板继续挂法人）：`{"onuType":"Internet","services":{"internet":{"svlan":1113,"cvlan":1,"uv":100,"scos":0,"ccos":0},"tr069":{"cvlan":1000,"uv":1000,"scos":6,"ccos":6}}}`。

## 8. 可靠性与安全

- **ctag**：连接级自增（如 `T000001`），响应严格按 ctag 配对；超时未回按失败留痕（原始报文入日志可 grep）；
- **错误判定**：只认完成码 + EN 码；ENDESC 仅入日志；"资源已存在"类 EN 码值各命令不一，统一用 LST 预检实现幂等，不赌错误码；
- **限速**：单连接串行天然 ≤10 条/秒；存量批量场景在 executor 层加节流（默认 8 QPS）；
- **凭据**：UN/PWD 走 `provision_nms.pass_cipher` 加密存储或环境变量注入，**禁止入代码库**；
  ⚠️ xlsx 含真实口令（`Boss@123`），`docs/books/` 当前未入库（untracked）——建议 `.gitignore` 该目录或脱敏后再 commit；
- **观测**：失败路径一律 `[provision-tl1] EXEC FAILED task=<no> ctag=<t> EN=<code> ENDESC=<desc>` 级可 grep 日志；
  连接事件（RECONNECT/LOGIN FAILED/SHAKEHAND TIMEOUT）同样留痕，禁止静默吞错；
- **协议硬化 (P2)**：TCP → SSL（对齐文档建议与仓库密钥策略）。

## 9. 测试与验收

1. **codec 单测**：用 PDF 全部响应样例做 golden（LOGIN 成功、DENY+EN、多块查询 `>` 续块、DELAY）；
2. **session 单测**：fake conn 注入（沿用 TelnetExecutor.Dial 注入模式）；
3. **`cmd/tl1sim`（新增）**：U2000 仿真器，监听 13027，镜像文档响应格式（头行/`M ctag COMPLD`/EN=0），
   支持注入 DENY/DELAY/重复 ONU 错误，记录收到的指令供断言（oltsim 协议不同，不扩展它，独立子命令）；
4. **102 集成**：tl1sim + provisioner(DRIVER=tl1) + bossctl 驱动订单 6→11 环节全链路，
   断言 provision_logs SUCCESS 与 tl1sim 收到的指令序列；验收后 acc_ 数据清理；
5. **真实环境证据边界**：拿到局方 U2000 13027 访问权之前，一切验证仅到 tl1sim 层，
   对外交付物明确标注"未在真实 U2000 验证"（对齐 2026-09-04 real-environment-evidence 制度）。

## 10. 分期落地

| 期 | 内容 | 验收口径 |
|:---|:-----|:---------|
| P0 | tl1 包 codec/session/client + 单测；cmd/tl1sim；config + driver 开关 | `go test ./internal/domain/provision/... ./cmd/tl1sim/...`；tl1sim 起后 client 脚本完成 LOGIN+SHAKEHAND+LST 往返 |
| P1 | 迁移（§7 三项）；ParamResolver + executor；环节7/10/11 接线；102 e2e | 102 上订单走通 6→11，provision_logs 有真实 TL1 执行留痕；make check 全绿 |
| P2 | 退订/换机（DEL-ONU/CFG-ONU）；告警口 13028（SUBSCRIBE → notify 域/device.alarm）；SSL | 销户订单网元侧无残留（LST-ONU 查无）；告警入 device.alarm |

## 11. 开放问题（实施前需确认）

1. **U2000 可达性**：xlsx 里 `172.16.16.2` 是 OLT 侧标识；U2000 服务器真实地址、13027 端口、防火墙 ACL 需局方开通；
2. **License**：TL1 业务发放 License（LNSDNELCR86 等）是否已购、额度多少；
3. **ONUNO 分配策略**：每 PON 口独立 0~127（1G）/0~255（10G），回收时机=DEL-ONU 成功后；是否需要预留段避开手工工单；
4. **VLAN 规划来源**：SVLAN/CVLAN/UV 先由模板 content 承载（admin 手工配）；是否接 ODN 域自动规划，暂不做；
5. **模板集名称**（ONUTYPE，如 `Internet`）是 U2000 网管侧预配置的模板集，需与局方对齐清单后录入模板 content。
