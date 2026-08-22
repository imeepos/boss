-- 邀请奖励发券:invite_config 增加奖励券模板引用。
-- 语义:注册请求携带 inviteCode(邀请人手机号)且配置了 reward_template_id 时,
-- 注册成功后向邀请人按模板发一张券(source=INVITE)。

ALTER TABLE invite_config
    ADD COLUMN reward_template_id BIGINT REFERENCES coupon_templates (template_id);
