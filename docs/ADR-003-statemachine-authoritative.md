# ADR-003:状态机为唯一权威,Temporal 只做编排

## 背景
技术栈同时引入「定义驱动状态机(`internal/pkg/statemachine`)」与「Temporal 订单12环节编排」,
二者都涉及状态流转。若不划清边界,会出现两套事实源:状态机已拒绝的非法流转被 Temporal 侧推进,
或 Temporal 已补偿回退但状态机仍停留在旧状态,导致订单/资产生命周期漂移。

## 决策
1. **单事实源**:`internal/pkg/statemachine` 是「状态迁移是否合法」的唯一判定方
   (state/event/guard/next_state);任何状态推进前必须先过 `Machine.Transition`,被拒绝即失败。
2. **Temporal 只做编排/重试/补偿**:Activity 内部调用状态机做合法性判定,判定通过才改库;
   Temporal 负责失败重试、人工干预、Saga 补偿的顺序,不自行定义"合法状态"。
3. `statemachine` 保持**纯函数、无 I/O**,可同时被领域层与 Temporal Activity 复用;
   状态表(state -> event -> next + guards)由各领域初始化时注入,不在包内硬编码。

## 结果
- 状态合法性单点判定,规避双重事实源。
- 阶段5 人工闭环与阶段7 自动化共用同一状态机,「人工 → 自动」升级只改 Activity 实现、不改状态语义。

## 代价 / 备选
- 若完全不用 Temporal,状态机 + outbox 表也能闭环(技术栈方案第六章备选1);
  本决策保留该降级路径:状态机独立可用,不绑死 Temporal。
