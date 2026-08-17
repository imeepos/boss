# ADR-002:地址层级以 ltree `path` 为唯一权威

## 背景
阶段1 建立「市级 → 区级 → 街道 → 小区 → 楼栋」层级,后续所有业务(资产、资源、订单、GIS 八级钻取)
都以它为统一地址基准。`addresses` 表同时存在 `path`、`level`、`parent_id` 三个表达层级的信息,
若不约定权威来源,会出现互相矛盾的数据(如 level=3 但 path 有 5 层),直接破坏阶段8 八级钻取的准确性。

## 决策
1. **`path`(LTREE)是唯一权威**:
   - `level` 是查询冗余列,由 `nlevel(path)` 派生,受 `CHECK (nlevel(path) = level)` 约束;
   - `parent_id` 是派生列,由 `subpath(path, 0, -1)` 反查父节点 id 得到,应用层不手填。
2. 层级范围受 `CHECK (level BETWEEN 1 AND 5)` 约束,禁止非法层级。
3. `path` 唯一约束兜底重复层级路径。

## 结果
- 下钻/回退、按层统计只依赖 `path` 单一来源,消除冗余列漂移。
- 应用层插入地址时只提交 `path + name`,其余派生列由服务侧统一计算(详见 `user.Service.ImportAddresses` 契约)。

## 代价 / 备选
- 触发器自动派生 `parent_id` 会引入额外复杂度,本轮以「服务层统一计算 + CHECK 兜底」落地;
  若后续发现绕过服务层的直接 SQL 写入带来漂移风险,再补触发器。
