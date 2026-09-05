# 2026-09-06 资产/标签数据质量闸门(000184)与写侧事务化

> 触发:用户要求调研真实 BOSS 系统资产与标签的成熟设计并打磨本系统。
> 调研全文见 docs/design/asset-tag-research-mature-designs.md;102 真库实证(2026-09-06):
> assets 206 条中 1 条 status=''/type=''(#190,e2e fixture 残留),tags 208 条中 2 条 status=''
> (#161/#186,同因);asset_lifecycles 仅 6 行 vs 206 资产(建档不留痕);asset_assignments 0 行
> (只有 GET 无 POST 的死台账);全系统无任何标签解绑路径。

## 一、裁定(不可逆)

1. **状态枚举 DB CHECK 兜底**:assets.status 四态、tags.status 三态、assets.type 非空串。
   理由:terms.md §4 枚举早已是契约权威,但只有应用层校验,fixture/运维 SQL 可绕过;
   成熟 CMDB/ITAM(蓝鲸唯一校验、云厂商标签写入端硬校验)均以存储层约束兜底。
2. **存量清洗口径**:非法值按默认态归一(status=''→IN_STOCK/UNBOUND,type=''→ONU),
   不做业务推断;'光猫'与'ONU'并存是存量口径,P1 型号字典落地时再统一。
3. **建档即留痕**:CreateAsset 与采购入库确认(ConfirmReceipt)同事务落 asset_lifecycles 首行。
   理由:生命周期轨迹从建档起完整是 ITAM 底线;此前 206 资产仅 6 条轨迹,台账审计不可用。
4. **写侧事务化**:CreateAsset/CreateTag 包 tx(建主档+轨迹+双绑回填同生共死),
   双绑冲突 ErrBindingConflict 时整单回滚,杜绝'资产已建但绑定失败'的中间态孤儿。
   历史教训:000158 修复的是双绑不一致,本次修的是半成品事务。

## 二、放弃的方案

- **type 收敛为封闭枚举 CHECK**:会把存量合法值('光猫')与未来类型卡死,应走型号字典
  (asset_models,Snipe-IT Fieldset 思路)而非硬枚举;本次仅拦空串。
- **存量资产回填'开账轨迹'行**:DEPLOYED 资产的建档时刻≠部署时刻,回填会伪造历史;
  成熟系统同样只保证'上线后'轨迹完整。轨迹覆盖从本迁移起算。
- **SetAssetStatus 内部强插轨迹行**:换新完成 handler 已在调用侧落轨迹,域层再落会双写;
  收敛为单一转移入口是 P1(装机/拆机资产联动)的范围。

## 三、验收

- go build/vet/gofmt 全绿;internal/domain/asset + procurement 单测全 PASS
  (11 处 mock 期望序列补 ExpectBegin/Commit/Rollback + lifecycle 首行)。
- 102 应用 000184 后:三约束存在、脏行归零(UPDATE 幂等,重复执行无副作用)。
- 后续回归:POST /provision/assets 带冲突 tagId → 40900 且资产未落库(事务回滚)。

## 四、遗留(P1/P2,见调研报告路线图)

- G1 装机/拆机不联动资产台账(扫码 LINKED 不置 DEPLOYED;无 SCRAPPED 路径)——最大漂移源。
- G5 标签无解绑/回收闭环(换新/报废后标签死绑)。
- G6 无 SN/MAC/LOID 设备身份与型号字典。
- G7 列表接口全量返回无过滤分页;asset_assignments 死表。

## 五、参考

- docs/design/asset-tag-research-mature-designs.md(四路业界调研综合)
- docs/notes/adopted/2026-08-27-asset-tag-bidirectional-binding.md(前序:双绑回填修复)
- migrations/000158_asset_tag_bidirectional_unique.up.sql(DB 兜底前哨)