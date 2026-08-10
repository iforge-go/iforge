-- 0002_normalize_collaborator_roles: 将 collaborator 表角色值统一为小写命名
-- 与 PMS 模块角色命名对齐：admin / member / viewer
-- 同时兼容历史脏数据：可能存在的 WRITE/READ 字面量（SSH/LFS 旧逻辑写入）
-- 注意：SQLite/MySQL/PostgreSQL 均支持 LOWER()，幂等可重复执行。
UPDATE collaborator SET role = 'admin'  WHERE LOWER(role) = 'admin';
UPDATE collaborator SET role = 'member' WHERE LOWER(role) IN ('developer', 'write');
UPDATE collaborator SET role = 'viewer' WHERE LOWER(role) IN ('guest', 'read');
