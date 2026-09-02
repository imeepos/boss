# TL1 对接详细设计（2026-09-02，方案已确认）

> 上游方案：`docs/plan/tl1-northbound-integration-plan.md`。本文是可施工级设计：接口签名、取数路径、
> 编排流程、DDL、仿真器与验收命令。协议事实源 = PDF §10/§11/§12/§13/§15。

## 1. 代码分类（哪个是测试代码）

| 分类 | 位置 | 说明 |
|:-----|:-----|:-----|
| **生产代码** | internal/domain/provision/tl1/（7 文件）、迁移 000178、cmd/provisioner 装配、internal/app resolver | 上线运行 |
| **测试代码** | cmd/tl1sim/（U2000 仿真器，只跑开发/验收环境）、各包 *_test.go、102 e2e 脚本 | 不进生产部署 |
| 既有测试设施 | cmd/oltsim/ | 旧 TelnetExecutor 的仿真器，保留不动 |

## 2. 包结构与行数预算（红线 300/文件）

```text
internal/domain/provision/tl1/
  codec.go      ~180 行  命令构建 + 响应解析（纯函数,零依赖）
  wire.go       ~120 行  连接读写循环:帧读取(到终止符 ;/>),超时,续块拼接
  session.go    ~200 行  会话状态机:dial→LOGIN→ready,ctag 配对,SHAKEHAND,LOGOUT
  manager.go    ~120 行  端点级会话持有者:懒建连,断线退避重连,Do 包装
  client.go     ~180 行  业务命令:参数结构体→Command→Do→判定→类型化错误
  executor.go   ~200 行  provision.Executor 实现:StageEvent 编排 + 幂等分支
  errors.go      ~40 行  错误哨兵与 EN 码归类
```

## 3. 核心类型与接口

### 3.1 codec.go —— TL1 消息层（纯函数）

```go
type KV struct{ K, V string }

// Command 一条 TL1 命令(target 恒空,按文档占用冒号位)。
type Command struct {
    Verb    string // "ADD-ONU" / "LST-ONUSTATE" ...
    Access  []KV   // OLTID=...,PONID=NA-0-7-5
    Payload []KV   // AUTHTYPE=LOID,ONUID=...
}

// Build 序列化为 "VERB::K=V,K=V:ctag::K=V;\n";
// 值过白名单 [A-Za-z0-9 ()_+-./\\],违规即 error(fail fast,防注入脏参数到网管)。
func Build(c Command, ctag string) ([]byte, error)

// Response 解析后的响应;查询类 Rows 为表格行(attrib→value),操作类 Rows 为空。
type Response struct {
    SID, CTag, Completion string // COMPLD/DELAY/DENY/PRTL/RTRV
    EN     int
    ENDESC string
    Rows   []map[string]string
    Raw    string // 原始报文,失败留痕用
}

// Parse 解析完整报文(调用方保证已按终止符收全,含 > 续块拼接后的全文)。
// 跳过头行("HW_ip date")、"Title = ..."行与 "---" 分隔行;查询表按 TAB 切 attrib/value。
func Parse(raw string) (*Response, error)
```

### 3.2 session.go —— 会话层

```go
type Config struct {
    Addr        string        // host:13027
    Network     string        // "tcp"(P2 扩展 "ssl")
    User, Pass  string
    Keepalive   time.Duration // 0=3min(文档 10min 空闲断连,取 1/3)
    CmdTimeout  time.Duration // 0=120s(文档 2min 内响应)
    DialTimeout time.Duration // 0=10s
}

// Dial 建连并 LOGIN;鉴权失败返回 ErrAuth(带 EN/ENDESC,不重试)。
func Dial(ctx context.Context, cfg Config) (*Session, error)

// Do 串行执行一条命令:自增 ctag → 写 → 读到终止符(> 则续读拼接) → Parse。
// Completion=DELAY:继续等下一条同 ctag 最终响应,CmdTimeout 兜底。
// 连接断/读超时返回 ErrConnBroken。SHAKEHAND 后台 ticker 与 Do 共用写锁,空闲才发。
func (s *Session) Do(ctx context.Context, c Command) (*Response, error)

// Close LOGOUT(尽力)+关连接。
func (s *Session) Close() error
```

### 3.3 manager.go —— 端点层

```go
// Manager 持有单端点会话(懒建连);Do 遇 ErrConnBroken → 退避重连一次。
type Manager struct{ /* cfg, mu, sess, lastErrAt */ }
func NewManager(cfg Config) *Manager
func (m *Manager) Do(ctx context.Context, c Command) (*Response, error)
// WithSession 多命令原子段:会话中途断开,整段闭包重跑(配合 executor 先查后写=幂等)。
func (m *Manager) WithSession(ctx context.Context, fn func(*Client) error) error
```

### 3.4 client.go —— 业务命令层

```go
type Client struct{ m *Manager }
// 写类命令不做内部重发(非幂等),重试由 executor 编排层"先查后写"实现。
func (c *Client) LstONU(ctx, oltid, ponid, idType, id string) ([]map[string]string, error) // 空=不存在
func (c *Client) AddONU(ctx, p AddONUParams) error
func (c *Client) AddPONVLAN(ctx, p PONVLANParams) error
func (c *Client) LstPONVLAN(ctx, oltid, ponid, idType, id string) ([]map[string]string, error)
func (c *Client) LstONUState(ctx, oltid, ponid, idType, id string) (ONUState, error)

type ONUState struct{ AdminState, OperState, CFGSTAT string }
// OperState 枚举(PDF §15.4.4): UP 在线 / Power-Off 掉电 / LOS 断纤 / other
// CFGSTAT 枚举: INITIAL 初始 / NORMAL 正常 / FAILED / CONFIG
```

### 3.5 executor.go —— provision.Executor 实现

```go
type ParamResolver interface {
    // Resolve 一次取齐全部参数;缺任一项返回带明确原因的 error(任务 FAILED 可诊断)。
    Resolve(ctx context.Context, t provision.Task) (Params, error)
}

type Endpoint struct{ Host string; Port int; User, Pass string }

type Params struct {
    Endpoint        Endpoint
    OLTID, PONID    string // PONID 形如 NA-0-7-5
    AuthType, ONUID string // 缺省 LOID + lo_accounts.loid
    ONUNo           int    // 0~127
    ONUType, Desc   string // ONUType=U2000 模板集名;Desc=PRV-<orderNo>(幂等对账键)
    Services        []PONVLANParams
}

func NewExecutor(m *Manager, r ParamResolver) *Executor
func (e *Executor) Exec(ctx context.Context, t provision.Task) error // 按 StageEvent 分派
```

## 4. 参数解析链路（resolver 实现放 internal/app/provision_tl1.go）

```text
provision_tasks.id
 ├─→ orders(order_no, customer_id, legal_entity_id)
 ├─→ lo_accounts(customer_id)          → loid(环节6 已建;缺=FAIL "lo account missing")
 ├─→ provision_templates(template_id)  → content JSONB(onuType + services[])
 └─→ ports(order_id AND status IN (RESERVED,USED)) → 恰 1 行
       ├─→ pon_frame/pon_slot/pon_port  → PONID="NA-%d-%d-%d"(缺列=FAIL "port missing PON positioning")
       ├─→ onu_no NULL ? 分配(下) : 复用
       └─→ resources(id,type=OLT)       → nms_oltid → OLTID(缺=FAIL "OLT missing nms_oltid")
provision_nms(legal_entity_id UNIQUE)   → host/port/user/pass(缺行=FAIL "no NMS endpoint for entity %d")
```

ONUNO 分配器（幂等，同事务回写 ports.onu_no）：

```sql
INSERT INTO pon_onu_alloc(olt_resource_id,pon_frame,pon_slot,pon_port,next_no)
VALUES($1,$2,$3,$4,0)
ON CONFLICT (olt_resource_id,pon_frame,pon_slot,pon_port)
DO UPDATE SET next_no = pon_onu_alloc.next_no + 1
RETURNING next_no;  -- 首次 0,其后 1,2,...
```

## 5. 环节编排（写类命令一律先查后写,重试安全）

### preConfigOLT（环节7）

```text
params ← resolver.Resolve(task)
1. LstONU(LOID, loid)
   非空 → [provision-tl1] IDEMPOTENT SKIP(ONU-EXISTS) task=... → 跳 2
   空   → AddONU(AUTHTYPE=LOID, ONUID=loid, ONUNO=n, DESC=PRV-<orderNo>, ONUTYPE=<模板集>)
2. for svc in params.Services:
   LstPONVLAN(...) 命中 DESC==PRV-<orderNo>-<svc.Name> → SKIP
   未命中 → AddPONVLAN(SVLAN/CVLAN/UV/SCOS/CCOS 按模板;TR069 单层不带 SVLAN)
3. 任一步 DENY → return err(Daemon.FailTask 留痕含 EN/ENDESC/ctag)
```

### activateUser（环节10）

```text
LstONUState(LOID, loid):
  ONU 不存在                     → FAILED "onu not provisioned"(环节7 未成功的护栏)
  OperState=UP 且 CFGSTAT∈{NORMAL,CONFIG} → 成功
  OperState∈{Power-Off,LOS,other} → FAILED(装维布好 ONT 后重试),不伪造成功
```

### notifyActivation（环节11）

```text
LstPONVLAN: 全部 services 按 DESC=PRV-<orderNo>-<name> 命中 → SUCCESS 落回调
缺任一条 → FAILED(订单留 INSTALLING 可重试)
```

## 6. 响应判定与错误分类

| 情形 | 判定 | 处理 |
|:-----|:-----|:-----|
| 成功 | COMPLD 且 EN=0 | 继续 |
| 已存在 | 写前预检查出 | SKIP 留痕,不算错 |
| 业务失败 | DENY 且 EN!=0 | FAILED 留痕,可重试 |
| 鉴权失败 | LOGIN DENY | 不重试;`[provision-tl1] LOGIN FAILED EN=%d` 告警日志,Manager 熔断 5min |
| 延迟 | DELAY | 继续等同 ctag 最终响应,CmdTimeout 兜底 |
| 连接断 | EOF/读超时 | ErrConnBroken → Manager 重连;多命令段 executor 整段重跑(先查后写幂等) |
| 报文残缺 | Parse 失败 | FAILED + Raw 全文入日志 |

ENDESC 只进日志不当判定键;EN 码语义表随真实联调补进 errors.go 注释。
失败路径日志格式(可 grep):`[provision-tl1] EXEC FAILED task=<no> ctag=<t> cmd=<verb> EN=<code> ENDESC=<desc>`。

## 7. 配置与开关

```text
BOSS_PROVISION_DRIVER=log|telnet|tl1        # 默认 log(维持现状,零影响升级)
BOSS_PROVISION_TL1_ADDR=192.168.x.x:13027   # 环境兜底;生产以 provision_nms 行为准
BOSS_PROVISION_TL1_USER=...
BOSS_PROVISION_TL1_PASS=...
```

取数优先级:provision_nms(legal_entity_id) > 环境变量。dev/102 验收用环境变量连 tl1sim。

## 8. 迁移 000178（提案号;实施前重跑占号核查）

```sql
BEGIN;
CREATE TABLE provision_nms (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL UNIQUE REFERENCES legal_entities(id),
    host            VARCHAR(128) NOT NULL,
    port            INTEGER NOT NULL DEFAULT 13027,
    protocol        VARCHAR(8) NOT NULL DEFAULT 'tcp',  -- tcp(P2 扩 ssl)
    username        VARCHAR(32) NOT NULL,
    pass_cipher     TEXT NOT NULL,                      -- 复用 config_secrets 加解密 helpers
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE resources ADD COLUMN IF NOT EXISTS nms_oltid VARCHAR(128); -- OLT 行:U2000 侧标识
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_frame SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_slot  SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS pon_port  SMALLINT;
ALTER TABLE ports ADD COLUMN IF NOT EXISTS onu_no    SMALLINT;  -- NULL=未分配
CREATE TABLE pon_onu_alloc (
    olt_resource_id BIGINT NOT NULL REFERENCES resources(id),
    pon_frame SMALLINT NOT NULL,
    pon_slot  SMALLINT NOT NULL,
    pon_port  SMALLINT NOT NULL,
    next_no   SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (olt_resource_id, pon_frame, pon_slot, pon_port)
);
COMMIT;
```

配套契约同步(同提交):fields.md §下发 增列;domain-map.md provision 域增 tl1 子包与 app resolver。

## 9. cmd/tl1sim(测试代码,U2000 仿真器)

```text
flags: -addr 127.0.0.1:13027  -user admin  -pass admin
       -onu-oper-state UP|Power-Off|LOS   (环节10 负路径注入)
       -delay-ms 0                        (DELAY 注入)
       -deny-next ADD-ONU                 (单次故障注入)
       -record /tmp/tl1sim.jsonl          (指令留痕,供 e2e 断言)
state: onus{oltid|ponid|onuid}, svcports{oltid|ponid|onuid|desc}, per-conn loggedIn
行为:  LOGIN 校验 UN/PWD(错→DENY EN=76546031);未登录命令一律 DENY;
       ADD-ONU 重复→DENY;ADD-PONVLAN 要求 ONU 已注册;
       LST-ONU/LST-ONUSTATE 出表格式(含 --- 分隔行, TAB 对齐);
       SHAKEHAND/LOGOUT 按 COMPLD;响应模板逐字节对齐 PDF 示例(头行缩进 3 空格/M ctag/EN=0)。
```

## 10. 测试矩阵与验收命令

| 层 | 用例 | 验收命令 |
|:---|:-----|:---------|
| codec | PDF 样例 golden:LOGIN ok/deny、ADD ok/deny、LST 表格、> 续块、DELAY | go test ./internal/domain/provision/tl1/ -run TestCodec |
| session | fake conn:登录序、ctag 配对、SHAKEHAND 触发、断连 ErrConnBroken | go test ./internal/domain/provision/tl1/ -run TestSession |
| client+executor | 进程内起 tl1sim:环节7 全流程(含重复幂等)、环节10 三态、DELAY | go test ./internal/domain/provision/tl1/ -run TestExecutor |
| resolver | PG 集成:取数链 + 缺数据四类 FAIL 文案 + ONUNO 分配幂等 | go test ./internal/app/ -run TL1Resolver |
| sim 自测 | 协议回归 | go test ./cmd/tl1sim/ |
| 102 e2e | tl1sim + provisioner(DRIVER=tl1) + bossctl 驱动订单 6→11;断言 provision_logs.SUCCESS + tl1sim.jsonl 含 ADD-ONU/ADD-PONVLAN×2;环节10 注入 Power-Off 时订单不推进 | scripts/verify-tl1-e2e.sh(P1 交付),跑完 acc_ 清理 |

## 11. 装配改动点

- cmd/provisioner/main.go:driver switch 增 case "tl1"(env 端点构造 Manager + app resolver);默认分支不变;
- internal/app/provision_tl1.go:NewTL1ParamResolver(pool *pgxpool.Pool) tl1.ParamResolver;
- 不动 order/aaa/provision 域现有文件(Task 模型零变更,resolver 外部聚合,无导入环)。

## 12. 实施任务卡(P0+P1,每卡独立提交走 worktree 协议)

| # | 任务 | 验收 |
|:--|:-----|:-----|
| T1 | tl1 包 codec+wire+errors+单测 | go test ./internal/domain/provision/tl1/ |
| T2 | session+manager+单测(fake conn) | 同上 -run TestSession 全绿 |
| T3 | cmd/tl1sim + client + executor + 进程内 e2e 单测 | go test ./cmd/tl1sim/ ./internal/domain/provision/tl1/ |
| T4 | 迁移 000178(up/down) + fields.md/domain-map 同步 | make check(含 contract-sync 拦截项) |
| T5 | app resolver + 装配 + env 开关 | go test ./internal/app/ -run TL1;本地 sim+provisioner 走通一单 |
| T6 | 102 e2e 脚本 + 真实验证边界声明 | scripts/verify-tl1-e2e.sh 在 102 全绿 |

真实 U2000 联调前,所有交付物标注"仅 tl1sim 验证"。
