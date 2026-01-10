-- ============================================================
-- RBAC 初始化脚本
-- 用于初始化角色、权限和角色权限绑定关系
-- ============================================================

-- 插入角色数据
INSERT INTO `roles` (`role_tag`, `name`, `description`, `created_at`, `updated_at`) VALUES
('user_basic', '基本用户', '默认角色', NOW(), NOW()),
('user_active', '活跃用户', '活跃度达标解锁', NOW(), NOW()),
('user_verified', '认证用户', '完成校内身份认证', NOW(), NOW()),
('operator', '运营', '运营', NOW(), NOW()),
('admin', '管理', '管理', NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    `name` = VALUES(`name`),
    `description` = VALUES(`description`),
    `updated_at` = NOW();

-- 插入权限数据
INSERT INTO `permissions` (`permission_tag`, `name`, `description`, `created_at`, `updated_at`) VALUES
-- 基础用户权限
('user.get', '用户资料查看', '', NOW(), NOW()),
('user.update', '用户资料修改', '', NOW(), NOW()),
('oss.token.get', '获取OSS Token', '', NOW(), NOW()),
('review.create', '发布点评', '', NOW(), NOW()),
('review.get.self', '查看本人点评', '', NOW(), NOW()),
('coursetable.get', '查看课表', '', NOW(), NOW()),
('coursetable.class.search', '搜索班级', '', NOW(), NOW()),
('coursetable.class.update.own', '更新本人班级', '', NOW(), NOW()),
('coursetable.class.update.all', '管理员更新班级', '', NOW(), NOW()),
('coursetable.update', '更新个人课表', '', NOW(), NOW()),
('failrate', '挂科率查询', '', NOW(), NOW()),
('point.get', '积分查看', '', NOW(), NOW()),
('point.spend', '积分消费', '', NOW(), NOW()),
('statistic.get', '统计查看', '', NOW(), NOW()),
('contribution.get', '投稿查看', '', NOW(), NOW()),
('contribution.create', '投稿创建', '', NOW(), NOW()),
('countdown', '倒数日', '', NOW(), NOW()),
('studytask', '学习任务', '', NOW(), NOW()),
('material.get', '资料查看', '', NOW(), NOW()),
('material.rate', '资料评分', '', NOW(), NOW()),
('material.download', '资料下载', '', NOW(), NOW()),
('material.category.get', '资料分类查看', '', NOW(), NOW()),
('question', '刷题访问', '', NOW(), NOW()),
('notification.get', '通知后台查看', '', NOW(), NOW()),

-- 管理权限
('review.manage', '点评管理', '', NOW(), NOW()),
('coursetable.manage', '课表管理', '', NOW(), NOW()),
('hero.manage', '英雄榜管理', '', NOW(), NOW()),
('config.manage', '配置管理', '', NOW(), NOW()),
('point.manage', '积分管理', '', NOW(), NOW()),
('contribution.manage', '投稿管理', '', NOW(), NOW()),
('notification.create', '通知创建', '', NOW(), NOW()),
('notification.publish', '通知发布', '', NOW(), NOW()),
('notification.update', '通知更新', '', NOW(), NOW()),
('notification.approve', '通知审核', '', NOW(), NOW()),
('notification.schedule', '通知排期', '', NOW(), NOW()),
('notification.pin', '通知置顶/撤销', '', NOW(), NOW()),
('notification.delete', '通知删除', '', NOW(), NOW()),
('notification.publish.admin', '通知直发', '', NOW(), NOW()),
('notification.category.manage', '通知分类管理', '', NOW(), NOW()),
('feature.manage', '功能管理', '', NOW(), NOW()),
('user.manage', '用户管理', '', NOW(), NOW()),
('material.manage', '资料管理', '', NOW(), NOW()),
('s3.manage', 'S3管理', '', NOW(), NOW())
ON DUPLICATE KEY UPDATE 
    `name` = VALUES(`name`),
    `description` = VALUES(`description`),
    `updated_at` = NOW();

-- 清理并重建角色权限绑定关系
DELETE FROM `role_permissions` WHERE `role_id` IN (
    SELECT `id` FROM `roles` WHERE `role_tag` IN ('user_basic', 'user_active', 'operator')
);

-- 绑定基础用户权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`, `created_at`, `updated_at`)
SELECT 
    (SELECT `id` FROM `roles` WHERE `role_tag` = 'user_basic') as role_id,
    `id` as permission_id,
    NOW() as created_at,
    NOW() as updated_at
FROM `permissions`
WHERE `permission_tag` IN (
    'user.get', 'user.update', 'oss.token.get', 
    'review.create', 'review.get.self',
    'coursetable.get', 'coursetable.class.search', 'coursetable.class.update.own', 'coursetable.update',
    'failrate', 'point.get', 'point.spend',
    'contribution.get', 'contribution.create',
    'countdown', 'studytask',
    'material.get', 'material.rate', 'material.download', 'material.category.get',
    'question', 'notification.get'
);

-- 绑定活跃用户权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`, `created_at`, `updated_at`)
SELECT 
    (SELECT `id` FROM `roles` WHERE `role_tag` = 'user_active') as role_id,
    `id` as permission_id,
    NOW() as created_at,
    NOW() as updated_at
FROM `permissions`
WHERE `permission_tag` IN ('coursetable.class.update.all');

-- 绑定运营权限
INSERT INTO `role_permissions` (`role_id`, `permission_id`, `created_at`, `updated_at`)
SELECT 
    (SELECT `id` FROM `roles` WHERE `role_tag` = 'operator') as role_id,
    `id` as permission_id,
    NOW() as created_at,
    NOW() as updated_at
FROM `permissions`
WHERE `permission_tag` IN (
    'contribution.manage',
    'notification.get', 'notification.create', 'notification.publish',
    'notification.update', 'notification.approve', 'notification.schedule'
);

-- 查看初始化结果
SELECT '角色列表:' as info;
SELECT `id`, `role_tag`, `name`, `description` FROM `roles`;

SELECT '权限统计:' as info;
SELECT COUNT(*) as total_permissions FROM `permissions`;

SELECT '角色权限绑定统计:' as info;
SELECT r.`role_tag`, r.`name`, COUNT(rp.`id`) as permission_count
FROM `roles` r
LEFT JOIN `role_permissions` rp ON r.`id` = rp.`role_id`
GROUP BY r.`id`, r.`role_tag`, r.`name`
ORDER BY r.`id`;
