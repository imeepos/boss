-- 用户地址簿挂接地址层级树:user_addresses.address_path 存 ltree 字符串
-- (addresses.path 同构,不加强 FK:树节点允许改名/重整,路径字符串作弱引用)。
-- 空串 = 历史自由文本地址;App 级联选到街道/小区节点后回填(契约 AddressInfo.addressPath)。
BEGIN;
ALTER TABLE user_addresses ADD COLUMN address_path TEXT NOT NULL DEFAULT '';
COMMIT;
