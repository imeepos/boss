# W7 勘测采集与施工进度上报模型裁定(2026-09-07)

> P-INFRA-1 W7(审查 F5a/F5b 合并):勘测任务域(000223)+施工进度上报人代次(000224)。执行会话落。

## 裁定

1. **勘测任务域落 odn 包**(internal/domain/odn,survey_tasks/survey_task_reports):勘测目标是
   ODN 网格/设施规划输入,与施工项目同属投建链;状态机 PENDING→ACCEPTED→BACKFILLED,
   PENDING/ACCEPTED→CANCELLED;指派单仅指派师傅可接,未指派单任意在职师傅抢单占位
   (assigned_worker_id 接单即写,抢单并发由状态守卫 UPDATE 兜底);回填 append-only 只增不改不删,
   (task_id, client_msg_id) 唯一幂等,首次回填置 BACKFILLED,其后追加多勘测点不改状态。
2. **进度上报人代次 construction_progress.reporter_type**:ACCOUNT/WORKER 两代次,reported_by
   语义=代次域内 id(ACCOUNT→accounts.id / WORKER→workers.id),存量行(000213 起)默认 ACCOUNT
   零迁移;WORKER 上报仅限 BUILDING 项目(域层守卫),上报人姓名读路径联表回显不落列。
   师傅可见项目=BUILDING(施工中),与 admin 全态上报并存;进度记录只增不改(留痕口径,无更新端点)。
3. **GIS 衔接复用 /gis/odn-points**:entity 扩展 survey|progress(level 12/13),直接读
   survey_task_reports / construction_progress 非空坐标行,不建点位表、不复制数据;状态字段分别
   承载 suggestion(可装/需新建设施)与项目状态,前端图层开关一一对应。
4. **admin 呈现挂既有页面**:勘测任务=ODN 管理页新页签(menu:odn),进度卡=施工单详情内,
   GIS 图层=intel 页既有 odn 图层下拉扩两档;不新增菜单分组、不动中央注册文件(menu.def/App)。

## 放弃了什么

- 勘测任务独立 domain 包+独立菜单页:割裂投建链数据面,违背「不新增菜单」纪律,被否决。
- reported_by 拆双列(account_id/worker_id)或 worker 复用 accounts 命名空间:双列读路径全改、
  复用会与账号 id 冲突(横向越权先例),代次列最小改动且存量零迁移。
- 勘测回填单记录制(一任务一回填):多勘测点/多visit 场景丢事实,append-only 多记录+幂等键更贴现场。
- GIS 点位物化表或 GeoJSON 服务端拼装:与既有 odn-points 数值 bbox 机制重复,前端已自拼 FeatureCollection。
- 施工进度 worker 上报不限项目状态:违反「BUILDING 期现场可见性」口径,竣工/待开工项目无现场事实可报。

## 影响面

- terms.md §4(勘测状态机/建议枚举/代次语义)、fields.md 1.5.8b+1.5.16、domain-map.md odn 行、
  admin/odn.yaml、worker/{survey,construction}.yaml、intel.yaml。
- 师傅端 Android 勘测屏(列表/详情/接单/回填)复用既有屏模式;验收脚本 scripts/ops/w7-survey-acceptance.py。