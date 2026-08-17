# 垂直切片设计 · 客户→订单（internal/domain/{customer,order}）

> 版本 V1.0（2026-08-17）｜配套：terms.md / domain-map.md / fields.md
> 目标：固化「领域实体 + 服务接口 + 错误枚举」的编码范式，作为后续 12 域"套模板"的样例切片。
> 边界：只到**接口抽象层**（领域层只依赖接口，不依赖具体 DB/Redis），编译 + 单测验收，不碰真实连接与环境。

## 0. 为什么是"客户→订单"这个切片

1. 它覆盖 AG-02（CRM 含订单）域，是 12 环节状态机的起点（环节 1 下单、2 资源核查），跨域契约直观（CT-001 客户建档、CT-002 订单-资源核查）。
2. fields.md 已锁定 customer/order 的核心字段，不用再猜。
3. aaa 域已给出 interface + struct + errors 的范式，本切片对齐之，形成第二条"打样"。

## 1. 切片范围（明确不做，防止蔓延）

| 层 | 做 | 不做 |
|:---|:---|:-----|
| 实体 struct | Customer / Product / Order 核心字段（照 fields.md） | 账务/会话/配置等下游实体 |
| 服务接口 | CustomerService / OrderService / ResourceChecker | 具体 repo 实现、DB 读写 |
| 错误 | domain 级 ErrXxx | HTTP 映射（属 transport） |
| 状态机 | Order 状态 PENDING→RESERVED，用 statemachine 包定义 | 全 12 环节流转（本切片只到环节 2） |
| 契约 | 以 Go interface 表达 CT-001/CT-002 的域内调用点 | 真实跨进程 RPC/Kafka |

> 环节覆盖：本切片只落地「1 下单 → 2 资源核查 → 3 端口预占」的状态判定与接口骨架；
> 合同收费（4）及之后环节**留接口位、不实现**，防止前期硬编码后期返工。

## 2. 领域实体（照 fields.md，字段不另起别名）

### 2.1 customer 域

```go
// Customer 客户档案（REQ-CRM-001 一人一档；多联系人/站点属扩展，届时加 contact 子表）
type Customer struct {
    ID              int64
    Name            string
    Phone           string
    IdType          string // 身份证/护照/营业执照（enum 待 terms.md 补，先用 string）
    IdNo            string
    RealNameStatus  string // VERIFIED / PENDING
    ServiceStatus   string // ACTIVE / ARREARS / SUSPENDED
    AddressID       int64
    BrandID         int64  // 品牌横切维度
    RegionPath      string // ltree 区域维度
    CreatedAt       time.Time
}
```

### 2.2 order 域

```go
// Order 订单主表（REQ-ORD-002：12 环节可跟踪）
type Order struct {
    ID          int64
    OrderNo     string // ORD-20250817-001
    CustomerID  int64
    ProductID   int64
    AddressID   int64
    Stage       int8   // 1~12（见 terms.md 第 1 节）
    Status      string // PENDING/RESERVED/INSTALLING/DONE（见 terms.md 第 3 节）
    ChannelID   int64  // 渠道，REQ-ORD-006 必填不可改
    BrandID     int64
    RegionPath  string
    CreatedAt   time.Time
}

// StageLog 环节时间轴（order.html 的「环节时间轴」表）
type StageLog struct {
    ID        int64
    OrderID   int64
    Stage     int8
    Result    string // DONE / DOING / PENDING
    Retries   int
    FinishedAt *time.Time
}
```

## 3. 服务接口（只依赖接口，不依赖 DB）

### 3.1 customer 域

```go
// ErrCustomerNotFound 客户不存在。
var ErrCustomerNotFound = errors.New("customer: not found")

type CustomerService interface {
    // Create 建档（CT-001）：成功后事件「客户已建档」交订阅方消费（本切片只发领域事件值，不接 Kafka）。
    Create(ctx context.Context, c Customer) (id int64, err error)
    // Get 按 id 查客户档案。
    Get(ctx context.Context, id int64) (*Customer, error)
    // List 按过滤条件分页查询（管理后台 customer.html 列表）。
    List(ctx context.Context, q CustomerQuery) ([]Customer, error)
}

type CustomerQuery struct {
    NameKeyword string
    Phone       string
    Status      string
    Limit       int
    Offset      int
}
```

### 3.2 order 域

```go
// 订单域对"资源核查"只依赖接口，不 import resource 域实现（契约 CT-002）。
// ResourceChecker 由 resource 域（阶段4）提供、app 装配层注入；本切片以接口锁定边界。
type ResourceChecker interface {
    // Check 核查目标地址是否有空闲端口；返回 available=false 时给出可选方案。
    Check(ctx context.Context, addressID int64) (available bool, options []string, err error)
}

type OrderService interface {
    // Submit 下单（环节1）：校验客户/产品有效 → 建单 → 状态 PENDING、stage=1。
    Submit(ctx context.Context, req SubmitReq) (*Order, error)
    // CheckResource 资源核查（环节2）：调 ResourceChecker，成功则推进 stage 到 2。
    CheckResource(ctx context.Context, orderID int64) error
    // Reserve 端口预占（环节3）：当前仅做状态机 guard 骨架，status PENDING→RESERVED。
    Reserve(ctx context.Context, orderID int64) error
    // Track 订单跟踪（order.html 时间轴）：返回订单 + 环节日志。
    Track(ctx context.Context, orderID int64) (*Order, []StageLog, error)
}

type SubmitReq struct {
    CustomerID int64
    ProductID  int64
    AddressID  int64
    ChannelID  int64 // 必填，不可改（REQ-ORD-006）
    BrandID    int64
    RegionPath string
}
```

## 4. 状态判定（复用 internal/pkg/statemachine）

订单状态迁移（本切片最少集，后续环节按 terms.md 第 1 节逐步补 defs）：

```go
// order_status.go 内定义，注入 statemachine.New(defs)
var orderStatusDefs = []statemachine.Def{
    {From: "PENDING",  Event: "reserve", To: "RESERVED"},
    {From: "RESERVED", Event: "release", To: "PENDING"},
    // 后续环节按 12 环节补：charge→INSTALLING 等，本切片不预埋
}
```

> 关键纪律（ADR-003）：所有 Status 迁移必须过 `statemachine.Machine.Transition` 校验，
> 不允许在 Service 方法里直接改 Status 字段。

## 5. 事件（域内只产值，不接消息中间件）

| 事件 | 触发 | 值 | 引用 |
|:-----|:-----|:---|:-----|
| CustomerCreated | Create 成功 | customerID | 全案 4.5「客户已建档」 |
| OrderSubmitted | Submit 成功 | orderID | 全案 4.5「订单已提交」 |
| ResourceChecked | CheckResource 成功 | orderID, available | 全案 4.5「资源核查完成」 |

> 本切片以 `[]Event`（值对象）返回或回调接口暴露，不依赖 Kafka；接入 Kafka 属阶段5 传输层，届时换实现不换接口。

## 6. 文件与行数规划（遵守 AGENTS.md：文件≤300 行、函数≤30 行）

| 文件 | 内容 | 预计行数 |
|:-----|:-----|:---------|
| `internal/domain/customer/customer.go` | Customer struct + Query | ~40 |
| `internal/domain/customer/service.go` | CustomerService 接口 + Err | ~50 |
| `internal/domain/order/order.go` | Order/StageLog struct + SubmitReq | ~60 |
| `internal/domain/order/service.go` | OrderService + ResourceChecker 接口 + Err | ~70 |
| `internal/domain/order/status.go` | orderStatusDefs + 迁移辅助 | ~50 |
| `internal/domain/customer/service_test.go` | 表驱动单测（内存实现） | ~120 |
| `internal/domain/order/service_test.go` | 状态机 guard 单测 | ~120 |

## 7. 验收标准（单测即验收，不降级）

| 编号 | 断言 | 引用 |
|:-----|:-----|:-----|
| S-1 | Create 成功返回自增 id，Get 可回查同档 | REQ-CRM-001 |
| S-2 | Submit 校验客户不存在返回 ErrCustomerNotFound，不建单 | REQ-ORD-001 异常 |
| S-3 | Submit 成功后 order.Status=PENDING、Stage=1、ChannelID 已存 | REQ-ORD-002 / ORD-006 |
| S-4 | Reserve 前状态非 PENDING 时，statemachine 拒非法流转 | ADR-003 / OSS-002 |
| S-5 | CheckResource 依赖 ResourceChecker，available=false 返回 options 不下单预占 | CT-002 / REQ-OSS-001 |

> 未覆盖环节（4~12）与契约（CT-003 端口预占真实互斥、CT-004 收费）标记为「切片外」，不臆造实现，
> 待对应阶段落地；本切片只保证接口边界不堵路。

## 8. 切片后如何"套模板"（给后续域）

1. 照 `fields.md` 抄字段 → 写 struct（不另起别名）。
2. 照 aaa/customer 范式写 interface + errors（接口只依赖接口，不 import 他域实现）。
3. 有状态则用 `statemachine` 包，状态枚举引用 `terms.md`。
4. 事件只产值对象，不接中间件。
5. 单测走内存桩，表驱动，断言引用 REQ 编号。
