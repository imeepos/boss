# 本阶段收尾执行计划（AMap 完善 + 文档修正）

> 生成日期：2026-08-25 | 依据：本轮复盘识别项

## 1. 已完成确认（代码已入库，无需重做）

| 项目 | 提交 | 验证 |
|------|------|------|
| ETL 新鲜度派单闭环 | `3185f340` | 测试通过，loop 已注册 |
| 后台周期任务错峰 | `2678b5ad` | staggeredFirstRun 覆盖 cdr/etl_executor/patrol |
| 仪表盘日期边界 flake | `6b03f1c8` | 固定时钟测试通过 |
| 文档变更不触发部署 | `611e9f57` | 102 已部署 |

## 2. 阻塞项（并行会话所有，不触碰）

| 项目 | 原因 |
|------|------|
| importer-round3 收尾 | `/Users/imeepos/ext512/ymm-001/boss-round3` 有近期未提交修改，属于并行会话，按 AGENTS.md 红线禁触碰 |

## 3. 本轮执行项

### A. 高德地图暗色瓦片
- 亮色继续使用高德普通地图。
- 暗色改用 CartoDB Dark Matter。高德没有原生暗色瓦片，不对整个 OL 容器做 CSS filter，避免反转点位颜色和交互层。

### B. 高德地图 key 配置位
- 构建时从仓库根 `.env` 读取 `AMAP_KEY`，通过 Vite `define` 映射为 `import.meta.env.VITE_AMAP_KEY`。
- 只将公开的 `AMAP_KEY` 拼入浏览器瓦片请求；`AMAP_SECRET` 不下发浏览器，留给服务端 Web API 签名使用。
- 未配置 key 时保留无 key 公开瓦片回退，避免本地测试/无凭据环境直接失效。

### C. 过时任务文档修正
- 标记已完成项，避免后续会话重复开发。

### D. 300 行红线
- `cmd/bossctl/routes_admin.go` 虽超过 300 行，但首行是 `Code generated`。
- `scripts/check-contract-sync/filelen.go` 的 `isGenerated()` 已排除生成文件，因此不拆生成产物。

## 4. 执行顺序

1. A/B 在独立 worktree 完成，测试、typecheck、build 后提交并合并。
2. C 在独立 worktree 修正文档，检查契约同步后提交并合并。
3. 每次合并前重新同步 main；合并成功后清理 worktree 和分支。
