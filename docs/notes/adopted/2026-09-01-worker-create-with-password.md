# 2026-09-01 后台录入师傅 + 师傅端登录名=phone 裁定

日期: 2026-09-01

## 决策

1. admin 后台可直接录入师傅: `POST /api/admin/v1/workers` 必带 `password`（>=6 位,
   bcrypt 落 workers.password_hash）; `PUT /api/admin/v1/workers/{id}/password` 重置。
   师傅端 `mode=password` 登录同步接入（域内 VerifyPassword 比对哈希,不暴露 password_hash）。
2. 师傅端登录名裁定为 **phone**（手机号）。fields.md §7.1 旧文"staffNo 作为登录名"
   与 worker/auth.yaml（`required:[phone,mode]`）、师傅端 Android LoginScreen（phone+密码）
   三处实现均不符,按实现现况修正;staffNo 仅为工号展示/检索标识。

## why

师傅账号此前只能经"入驻注册→审核"创建且 password_hash 恒为空串,后台无录入入口,
师傅端密码登录后端恒拒("password mode not supported yet"),密码模式形同虚设。
登录名以实现现况为准：三处契约/代码均为 phone,改回 staffNo 需动师傅端 App+契约,
收益不抵改动面。

## 放弃了什么(被否决项)

- 登录名改 staffNo:与师傅端 App、worker/auth.yaml 现状冲突,需三端联动改版。
- 后台存明文密码/可找回:违背 000043 列注释"仅存哈希,不存明文";遗失走重置。
- 新建 worker_auth 独立表:workers.password_hash 列已存在(000043),无必要。

## 遗留

workers.phone 无唯一索引,应用层录入不查重（师傅端登录 findWorkerByPhone 取首个
在职命中）;若出现同号多在职师傅,登录归属性不确定,后续可加部分唯一索引
（`WHERE status=1`）+ 存量清洗,另立迁移。
