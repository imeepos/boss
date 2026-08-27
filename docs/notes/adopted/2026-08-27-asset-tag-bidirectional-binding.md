# 2026-08-27 资产台账 ↔ 电子标签双向绑定回填修复

> 依据: 用户报"资产台账和电子标签现在的数据都没有关联上",SQL 直查 boss-infra-postgres-1
> `boss.assets`/`boss.tags` 确认:资产 196 条只有 69 条有 `tag_id`(127 条孤儿),
> 标签 197 条有 192 条已 `bound_asset_id`。双向一致性只有 68 对,124 条 B 端孤儿
> (标签 `bound_asset_id` 已写但对应资产 `tag_id` 为空)+ 1 条 A 端孤儿
> (资产 `tag_id=124` 但标签 `bound_asset_id` 为空,实测 A-20260001)。

## 一、问题定级

行为缺口(契约层面):`assets.tag_id` 与 `tags.bound_asset_id` 必须互为镜像,但
**两个写入入口的回填链路都不完整**——

| 入口 | 文件 | 当前回填 | 漏洞 |
|------|------|----------|------|
| `POST /provision/assets`(CreateAsset) | `internal/domain/asset/pg_write.go:118` | `UPDATE tags SET bound_asset_id=…, status='BOUND' WHERE id=$tag AND bound_asset_id IS NULL` | 标签侧已先建并填了 `bound_asset_id` 时,UPDATE 影响 0 行,资产 `tag_id` 已写但标签侧无变化——回填静默成功,实际双向一致 |
| `POST /provision/tags`(CreateTag) | `internal/domain/asset/pg_write.go:37` | **完全无回填** | 标签 `bound_asset_id` 已写,但对应资产 `tag_id` 不会同步回填——B 端孤儿 |

历史 d397e40(2026-08-21,ISSUE.md 第5行)修复了 CreateAsset 侧的回填方向,但用了
`bound_asset_id IS NULL` 哑条件,实际业务流(e2e 先建标签并填 `bound_assetID=…,
Status="BOUND"` 绕过 40920)依然会落入"回填跳过"分支。d397e40 是局部修复,未覆盖
CreateTag 反向路径,本次补齐。

## 二、本次裁定(不可逆)

1. **两个写入入口都必须强制双向回填 + 冲突检测**:任一写入后,对应反向表必须
   同步;冲突(资产已被其他标签绑定 / 标签已被其他资产绑定)即返 ErrBindingConflict,
   不允许隐性双绑或静默孤儿。
2. **回填 SQL 去掉 `IS NULL` 哑条件,改为 `IS NULL OR = $expected`**:
   - 资产侧:`UPDATE assets SET tag_id = $tag WHERE id = $asset AND (tag_id IS NULL OR tag_id = $tag)`
   - 标签侧:`UPDATE tags SET bound_asset_id = $asset, status = 'BOUND' WHERE id = $tag AND (bound_asset_id IS NULL OR bound_asset_id = $asset)`
   - 这样 UPDATE 影响行数 0 必然意味着冲突(对方已被别人绑),可以可靠触发 ErrBindingConflict。
3. **冲突路径必须有可观测信号**(AGENTS.md "失败路径必须留有可观测信号"红线):
   写入返回 ErrBindingConflict 时同步 `slog.WarnContext("[asset] TAG BIND CONFLICT", …)`,
   含资产/标签 id + 资产码/标签号 + 冲突原因,便于排查时 grep 日志定位。
4. **新增 `ErrBindingConflict` sentinel** 与现有 `ErrForeignKeyViolation` 对齐;
   `CreateTag` 预绑定时先做资产存在性校验,缺失即 ErrForeignKeyViolation(置备侧孤儿防御)。

## 三、放弃的方案

- **数据库 unique index(bound_asset_id、tag_id)**:会让现有 124 条孤儿立即报错阻塞生产,
  必须先写迁移兜底清数据。本次修复仅做应用层双绑,不做 DB 约束——待存量孤儿清理后再加约束。
- **统一为单一写入路径(只允许先建资产或只允许先建标签)**:业务流 e2e 测试需要先建标签
  (扫码场景),强制单向会破坏 worker 端 `/tickets/:no/replace` 的换件流程(参 worker/asset.go)。
- **改为异步回填**(事务外另起 goroutine):会破坏事务原子性,资产/标签有一方失败时无法回滚。

## 四、验收

- 单元测试:新增 4 个 case 覆盖三种边界(资产不存在 / 资产已被绑 / 标签已被绑 / 正常回填),
  原 `TestPGStore_CreateTag`、`TestPGStore_CreateAsset`、`TestPGStore_CreateAsset_BackfillTagBinding`
  三个保留全部 PASS。
- 后端 `go test ./...` 全包通过(2026-08-27 wt-asset-tag-fix 本地实测)。
- `go build ./...`、`go vet ./...`、`gofmt -l` 全空。
- 102 实测(待 CI 部署后):调用 `POST /provision/tags` 预绑定资产应同步回写 `assets.tag_id`,
  反之亦然;冲突场景返 ErrBindingConflict + 日志 `[asset] TAG BIND CONFLICT`。

## 五、遗留(本任务不解决)

- 124 条历史 B 端孤儿(标签 `bound_asset_id` 已写但资产 `tag_id` 为空)需要单独数据清理任务,
  在新写入逻辑生效后再回填资产 `tag_id`(UPDATE assets SET tag_id = sub.tag_id FROM
  (SELECT id, bound_asset_id FROM tags WHERE bound_asset_id IS NOT NULL) sub WHERE
  assets.id = sub.bound_asset_id AND (assets.tag_id IS NULL OR assets.tag_id <> sub.id))。
  建议作为独立 chore 任务,本次不混合提交。
- 1 条 A 端孤儿(A-20260001,tag_id=124 但标签 bound_asset_id 为空)需人工核查:
  是标签真的没绑还是测试遗留,再决定回填或清空 tag_id。
- 未来增加 unique index `uq_tags_bound_asset_id` 与 `uq_assets_tag_id`(均非空时)
  来彻底杜绝应用层遗漏,但需先清存量孤儿。

## 六、参考

- ISSUE.md §后端 12 环节 2026-08-21(d397e40):同类局部修复历史
- docs/contract/fields.md §4.1 `assets.TagID` � `tag_id` 字段约定
- docs/contract/data-relations.md §资产层:`asset_batches → assets ↔ tags` 双向关系
