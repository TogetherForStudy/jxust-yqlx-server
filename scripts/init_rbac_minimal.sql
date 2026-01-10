-- ============================================================
-- RBAC 快速初始化脚本（测试用）
-- 最小化版本：仅创建E2E测试必需的角色和基本权限
-- ============================================================

-- 插入核心角色
INSERT IGNORE INTO `roles` (`role_tag`, `name`, `description`, `created_at`, `updated_at`) VALUES
('user_basic', '基本用户', '默认角色', NOW(), NOW()),
('admin', '管理', '管理', NOW(), NOW());

-- 插入核心权限（基础用户必需）
INSERT IGNORE INTO `permissions` (`permission_tag`, `name`, `description`, `created_at`, `updated_at`) VALUES
('user.get', '用户资料查看', '', NOW(), NOW()),
('user.update', '用户资料修改', '', NOW(), NOW()),
('review.create', '发布点评', '', NOW(), NOW()),
('coursetable.get', '查看课表', '', NOW(), NOW());

-- 绑定基础用户权限
INSERT IGNORE INTO `role_permissions` (`role_id`, `permission_id`, `created_at`, `updated_at`)
SELECT 
    (SELECT `id` FROM `roles` WHERE `role_tag` = 'user_basic') as role_id,
    `id` as permission_id,
    NOW() as created_at,
    NOW() as updated_at
FROM `permissions`
WHERE `permission_tag` IN ('user.get', 'user.update', 'review.create', 'coursetable.get');

-- 验证初始化结果
SELECT 'RBAC初始化完成' as status;
SELECT role_tag, name FROM roles WHERE role_tag IN ('user_basic', 'admin');
