#!/usr/bin/env python3
"""
GoJxust API E2E 测试脚本

使用方法:
    pip install httpx
    python scripts/e2e_test.py [--base-url <http://localhost:8080>] [--insecure 允许不安全的 HTTPS 连接（忽略证书错误）]

该脚本测试 GoJxust API 的主要端点，使用模拟微信登录获取授权。
"""

import httpx
import argparse
import sys
import uuid
from typing import Optional
from dataclasses import dataclass

# 默认配置
DEFAULT_BASE_URL = "http://localhost:8080"
API_PREFIX = "/api/v0"


@dataclass
class TestResult:
    """测试结果"""
    name: str
    passed: bool
    message: str


class E2ETestClient:
    """E2E 测试客户端"""

    def __init__(self, base_url: str, insecure: bool = False):
        self.base_url = base_url.rstrip("/")
        verify = not insecure
        if insecure and base_url.startswith("https"):
            print("⚠️  警告: 正在使用不安全的 HTTPS 连接，证书错误将被忽略。")
        self.client = httpx.Client(timeout=30.0, verify=verify)
        self.token: Optional[str] = None
        self.admin_token: Optional[str] = None
        self.results: list[TestResult] = []

    def _url(self, path: str) -> str:
        """构建完整 URL"""
        return f"{self.base_url}{API_PREFIX}{path}"

    def _headers(self, use_admin: bool = False) -> dict:
        """获取请求头"""
        token = self.admin_token if use_admin else self.token
        if token:
            return {"Authorization": f"Bearer {token}"}
        return {}

    def _record(self, name: str, passed: bool, message: str):
        """记录测试结果"""
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"{status}: {name} - {message}")
        self.results.append(TestResult(name, passed, message))

    # ==================== 认证相关 ====================

    def test_health_check(self) -> bool:
        """测试健康检查端点"""
        try:
            resp = self.client.get(f"{self.base_url}/health")
            passed = resp.status_code == 200 and resp.json().get("status") == "ok"
            self._record("健康检查", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("健康检查", False, str(e))
            return False

    def test_mock_wechat_login(self, test_user: str = "normal") -> Optional[str]:
        """测试模拟微信登录"""
        try:
            resp = self.client.post(
                self._url("/auth/mock-wechat-login"),
                json={"test_user": test_user}
            )
            if resp.status_code == 200:
                # 响应结构: {"StatusCode": 0, "StatusMessage": "Success", "RequestId": "...", "Result": {...}}
                result = resp.json().get("Result", {})
                token = result.get("token")
                if token:
                    self._record(f"模拟登录({test_user})", True, "获取 token 成功")
                    return token
                else:
                    self._record(f"模拟登录({test_user})", False, f"token 为空, body={resp.text}")
                    return None
            else:
                self._record(f"模拟登录({test_user})", False, f"status={resp.status_code}, body={resp.text}")
                return None
        except Exception as e:
            self._record(f"模拟登录({test_user})", False, str(e))
            return None

    def setup_auth(self) -> bool:
        """设置认证 token"""
        self.token = self.test_mock_wechat_login("basic")
        self.admin_token = self.test_mock_wechat_login("admin")
        return self.token is not None

    # ==================== 用户相关 ====================

    def test_get_profile(self) -> bool:
        """测试获取用户资料"""
        try:
            resp = self.client.get(
                self._url("/user/profile"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取用户资料", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取用户资料", False, str(e))
            return False

    def test_update_profile(self) -> bool:
        """测试更新用户资料"""
        try:
            resp = self.client.put(
                self._url("/user/profile"),
                headers=self._headers(),
                json={"nickname": "E2E测试用户"}
            )
            passed = resp.status_code == 200
            self._record("更新用户资料", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("更新用户资料", False, str(e))
            return False

    # ==================== 公开接口 ====================

    def test_get_reviews_by_teacher(self) -> bool:
        """测试按教师查询评价（公开）"""
        try:
            resp = self.client.get(
                self._url("/reviews/teacher"),
                params={"teacher_name": "测试老师"}
            )
            passed = resp.status_code == 200
            self._record("按教师查询评价", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("按教师查询评价", False, str(e))
            return False

    def test_get_config_by_key(self) -> bool:
        """测试获取配置（公开）"""
        try:
            resp = self.client.get(self._url("/config/test_key"))
            # 404 也算通过，因为配置可能不存在
            passed = resp.status_code in [200, 404]
            self._record("获取配置", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取配置", False, str(e))
            return False

    def test_list_heroes(self) -> bool:
        """测试获取英雄榜（公开）"""
        try:
            resp = self.client.get(self._url("/heroes/"))
            passed = resp.status_code == 200
            self._record("获取英雄榜", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取英雄榜", False, str(e))
            return False

    def test_get_notifications(self) -> bool:
        """测试获取通知列表（公开）"""
        try:
            resp = self.client.get(
                self._url("/notifications/"),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("获取通知列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取通知列表", False, str(e))
            return False

    def test_get_categories(self) -> bool:
        """测试获取分类列表（公开）"""
        try:
            resp = self.client.get(self._url("/categories/"))
            passed = resp.status_code == 200
            self._record("获取分类列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取分类列表", False, str(e))
            return False

    # ==================== 评价相关（需认证）====================

    def test_create_review(self) -> bool:
        """测试创建评价"""
        try:
            resp = self.client.post(
                self._url("/reviews/"),
                headers=self._headers(),
                json={
                    "teacher_name": "E2E测试老师",
                    "campus": "红旗校区",
                    "course_name": "E2E测试课程",
                    "content": "这是E2E测试创建的评价",
                    "attitude": 1
                }
            )
            passed = resp.status_code == 200
            self._record("创建评价", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("创建评价", False, str(e))
            return False

    def test_get_user_reviews(self) -> bool:
        """测试获取用户评价"""
        try:
            resp = self.client.get(
                self._url("/reviews/user"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取用户评价", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取用户评价", False, str(e))
            return False

    # ==================== 课程表相关（需认证）====================

    def test_get_course_table(self) -> bool:
        """测试获取课程表"""
        try:
            resp = self.client.get(
                self._url("/coursetable/"),
                headers=self._headers(),
                params={"semester": "2024-2025-1"}
            )
            # 如果用户未绑定班级返回 400，也算正常
            passed = resp.status_code in [200, 400]
            self._record("获取课程表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取课程表", False, str(e))
            return False

    def test_search_classes(self) -> bool:
        """测试搜索班级"""
        try:
            resp = self.client.get(
                self._url("/coursetable/search"),
                headers=self._headers(),
                params={"keyword": "计算机", "page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("搜索班级", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("搜索班级", False, str(e))
            return False

    # ==================== 挂科率相关（需认证）====================

    def test_search_fail_rate(self) -> bool:
        """测试搜索挂科率"""
        try:
            resp = self.client.get(
                self._url("/failrate/search"),
                headers=self._headers(),
                params={"keyword": "高数", "page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("搜索挂科率", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("搜索挂科率", False, str(e))
            return False

    def test_rand_fail_rate(self) -> bool:
        """测试随机挂科率"""
        try:
            resp = self.client.get(
                self._url("/failrate/rand"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("随机挂科率", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("随机挂科率", False, str(e))
            return False

    # ==================== 积分相关（需认证）====================

    def test_get_user_points(self) -> bool:
        """测试获取用户积分"""
        try:
            resp = self.client.get(
                self._url("/points/"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取用户积分", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取用户积分", False, str(e))
            return False

    def test_get_points_transactions(self) -> bool:
        """测试获取积分交易记录"""
        try:
            resp = self.client.get(
                self._url("/points/transactions"),
                headers=self._headers(),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("获取积分交易记录", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取积分交易记录", False, str(e))
            return False

    def test_get_points_stats(self) -> bool:
        """测试获取积分统计"""
        try:
            resp = self.client.get(
                self._url("/points/stats"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取积分统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取积分统计", False, str(e))
            return False

    # ==================== 投稿相关（需认证）====================

    def test_create_contribution(self) -> Optional[int]:
        """测试创建投稿"""
        try:
            resp = self.client.post(
                self._url("/contributions/"),
                headers=self._headers(),
                json={
                    "title": "E2E测试投稿",
                    "content": "这是E2E测试创建的投稿内容",
                    "categories": [1]
                }
            )
            passed = resp.status_code == 200
            contribution_id = None
            if passed:
                result = resp.json().get("Result", {})
                contribution_id = result.get("id")
            self._record("创建投稿", passed, f"status={resp.status_code}")
            return contribution_id
        except Exception as e:
            self._record("创建投稿", False, str(e))
            return None

    def test_get_contributions(self) -> bool:
        """测试获取投稿列表"""
        try:
            resp = self.client.get(
                self._url("/contributions/"),
                headers=self._headers(),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("获取投稿列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取投稿列表", False, str(e))
            return False

    def test_get_user_contribution_stats(self) -> bool:
        """测试获取用户投稿统计"""
        try:
            resp = self.client.get(
                self._url("/contributions/stats"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取用户投稿统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取用户投稿统计", False, str(e))
            return False

    # ==================== 倒数日相关（需认证）====================

    def test_create_countdown(self) -> Optional[int]:
        """测试创建倒数日"""
        try:
            resp = self.client.post(
                self._url("/countdowns/"),
                headers=self._headers(),
                json={
                    "title": "E2E测试倒数日",
                    "description": "测试描述",
                    "target_date": "2025-12-31"
                }
            )
            passed = resp.status_code == 200
            countdown_id = None
            if passed:
                result = resp.json().get("Result", {})
                countdown_id = result.get("id")
            self._record("创建倒数日", passed, f"status={resp.status_code}")
            return countdown_id
        except Exception as e:
            self._record("创建倒数日", False, str(e))
            return None

    def test_get_countdowns(self) -> bool:
        """测试获取倒数日列表"""
        try:
            resp = self.client.get(
                self._url("/countdowns/"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取倒数日列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取倒数日列表", False, str(e))
            return False

    def test_update_countdown(self, countdown_id: int) -> bool:
        """测试更新倒数日"""
        try:
            resp = self.client.put(
                self._url(f"/countdowns/{countdown_id}"),
                headers=self._headers(),
                json={"title": "E2E测试倒数日-已更新"}
            )
            passed = resp.status_code == 200
            self._record("更新倒数日", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("更新倒数日", False, str(e))
            return False

    def test_delete_countdown(self, countdown_id: int) -> bool:
        """测试删除倒数日"""
        try:
            resp = self.client.delete(
                self._url(f"/countdowns/{countdown_id}"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("删除倒数日", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("删除倒数日", False, str(e))
            return False

    # ==================== 学习任务相关（需认证）====================

    def test_create_study_task(self) -> Optional[int]:
        """测试创建学习任务"""
        try:
            resp = self.client.post(
                self._url("/study-tasks/"),
                headers=self._headers(),
                json={
                    "title": "E2E测试学习任务",
                    "description": "测试任务描述",
                    "due_date": "2025-12-31 23:59",
                    "priority": 2
                }
            )
            passed = resp.status_code == 200
            task_id = None
            if passed:
                result = resp.json().get("Result", {})
                task_id = result.get("id")
            self._record("创建学习任务", passed, f"status={resp.status_code}")
            return task_id
        except Exception as e:
            self._record("创建学习任务", False, str(e))
            return None

    def test_get_study_tasks(self) -> bool:
        """测试获取学习任务列表"""
        try:
            resp = self.client.get(
                self._url("/study-tasks/"),
                headers=self._headers(),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("获取学习任务列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取学习任务列表", False, str(e))
            return False

    def test_get_study_task_stats(self) -> bool:
        """测试获取学习任务统计"""
        try:
            resp = self.client.get(
                self._url("/study-tasks/stats"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取学习任务统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取学习任务统计", False, str(e))
            return False

    def test_get_completed_tasks(self) -> bool:
        """测试获取已完成任务"""
        try:
            resp = self.client.get(
                self._url("/study-tasks/completed"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取已完成任务", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取已完成任务", False, str(e))
            return False

    def test_update_study_task(self, task_id: int) -> bool:
        """测试更新学习任务"""
        try:
            resp = self.client.put(
                self._url(f"/study-tasks/{task_id}"),
                headers=self._headers(),
                json={"title": "E2E测试学习任务-已更新", "status": 2}
            )
            passed = resp.status_code == 200
            self._record("更新学习任务", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("更新学习任务", False, str(e))
            return False

    def test_delete_study_task(self, task_id: int) -> bool:
        """测试删除学习任务"""
        try:
            resp = self.client.delete(
                self._url(f"/study-tasks/{task_id}"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("删除学习任务", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("删除学习任务", False, str(e))
            return False

    # ==================== 管理员接口测试 ====================

    def test_admin_get_reviews(self) -> bool:
        """测试管理员获取评价列表"""
        try:
            resp = self.client.get(
                self._url("/reviews/"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员获取评价列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取评价列表", False, str(e))
            return False

    def test_admin_get_notifications(self) -> bool:
        """测试管理员获取通知列表"""
        try:
            resp = self.client.get(
                self._url("/admin/notifications/"),
                headers=self._headers(use_admin=True),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("管理员获取通知列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取通知列表", False, str(e))
            return False

    def test_admin_get_notification_stats(self) -> bool:
        """测试管理员获取通知统计"""
        try:
            resp = self.client.get(
                self._url("/admin/notifications/stats"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员获取通知统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取通知统计", False, str(e))
            return False

    def test_admin_search_heroes(self) -> bool:
        """测试管理员搜索英雄"""
        try:
            resp = self.client.get(
                self._url("/heroes/search"),
                headers=self._headers(use_admin=True),
                params={"q": "", "page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("管理员搜索英雄", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员搜索英雄", False, str(e))
            return False

    def test_admin_search_configs(self) -> bool:
        """测试管理员搜索配置"""
        try:
            resp = self.client.get(
                self._url("/config/search"),
                headers=self._headers(use_admin=True),
                params={"query": "", "page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("管理员搜索配置", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员搜索配置", False, str(e))
            return False

    # ==================== 功能白名单相关（需认证）====================

    def test_get_user_features(self) -> bool:
        """测试获取用户功能列表"""
        try:
            resp = self.client.get(
                self._url("/user/features"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            if passed:
                result = resp.json().get("Result", {})
                features = result.get("features", [])
                self._record("获取用户功能列表", True, f"features={features}")
            else:
                self._record("获取用户功能列表", False, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取用户功能列表", False, str(e))
            return False

    def test_admin_create_feature(self) -> Optional[str]:
        """测试管理员创建功能"""
        try:
            feature_key = f"beta_e2e_test_{uuid.uuid4().hex[:8]}"
            resp = self.client.post(
                self._url("/admin/features"),
                headers=self._headers(use_admin=True),
                json={
                    "feature_key": feature_key,
                    "feature_name": "E2E测试功能",
                    "description": "这是E2E测试创建的功能",
                    "is_enabled": True
                }
            )
            passed = resp.status_code == 200
            self._record("管理员创建功能", passed, f"status={resp.status_code}, key={feature_key}")
            return feature_key if passed else None
        except Exception as e:
            self._record("管理员创建功能", False, str(e))
            return None

    def test_admin_list_features(self) -> bool:
        """测试管理员获取功能列表"""
        try:
            resp = self.client.get(
                self._url("/admin/features"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员获取功能列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取功能列表", False, str(e))
            return False

    def test_admin_update_feature(self, feature_key: str) -> bool:
        """测试管理员更新功能"""
        try:
            resp = self.client.put(
                self._url(f"/admin/features/{feature_key}"),
                headers=self._headers(use_admin=True),
                json={
                    "feature_name": "E2E测试功能-已更新",
                    "description": "更新后的描述"
                }
            )
            passed = resp.status_code == 200
            self._record("管理员更新功能", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员更新功能", False, str(e))
            return False

    def test_admin_grant_feature(self, feature_key: str, user_id: int = 1) -> bool:
        """测试管理员授予功能权限"""
        try:
            resp = self.client.post(
                self._url(f"/admin/features/{feature_key}/whitelist"),
                headers=self._headers(use_admin=True),
                json={"user_id": user_id}
            )
            passed = resp.status_code == 200
            self._record("管理员授予功能权限", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员授予功能权限", False, str(e))
            return False

    def test_admin_list_whitelist(self, feature_key: str) -> bool:
        """测试管理员获取功能白名单"""
        try:
            resp = self.client.get(
                self._url(f"/admin/features/{feature_key}/whitelist"),
                headers=self._headers(use_admin=True),
                params={"page": 1, "page_size": 20}
            )
            passed = resp.status_code == 200
            self._record("管理员获取功能白名单", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取功能白名单", False, str(e))
            return False

    def test_admin_revoke_feature(self, feature_key: str, user_id: int = 1) -> bool:
        """测试管理员撤销功能权限"""
        try:
            resp = self.client.delete(
                self._url(f"/admin/features/{feature_key}/whitelist/{user_id}"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员撤销功能权限", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员撤销功能权限", False, str(e))
            return False

    def test_admin_get_user_features(self, user_id: int = 1) -> bool:
        """测试管理员查看用户功能权限"""
        try:
            resp = self.client.get(
                self._url(f"/admin/users/{user_id}/features"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员查看用户功能权限", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员查看用户功能权限", False, str(e))
            return False

    def test_admin_delete_feature(self, feature_key: str) -> bool:
        """测试管理员删除功能"""
        try:
            resp = self.client.delete(
                self._url(f"/admin/features/{feature_key}"),
                headers=self._headers(use_admin=True)
            )
            passed = resp.status_code == 200
            self._record("管理员删除功能", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员删除功能", False, str(e))
            return False


    # ==================== 幂等性测试 ====================

    def test_idempotency_create_review(self) -> bool:
        """测试幂等性：创建评价（带幂等性Key）"""
        try:
            idempotency_key = str(uuid.uuid4())
            headers = self._headers()
            headers["X-Idempotency-Key"] = idempotency_key
            
            resp = self.client.post(
                self._url("/reviews/"),
                headers=headers,
                json={
                    "teacher_name": "幂等性测试老师",
                    "campus": "红旗校区",
                    "course_name": "幂等性测试课程",
                    "content": "这是幂等性测试",
                    "attitude": 1
                }
            )
            passed = resp.status_code == 200
            self._record("幂等性-创建评价", passed, f"status={resp.status_code}, key={idempotency_key[:8]}...")
            return passed
        except Exception as e:
            self._record("幂等性-创建评价", False, str(e))
            return False

    def test_idempotency_duplicate_request(self) -> bool:
        """测试幂等性：重复请求返回缓存结果"""
        try:
            idempotency_key = str(uuid.uuid4())
            headers = self._headers()
            headers["X-Idempotency-Key"] = idempotency_key
            
            request_data = {
                "teacher_name": "重复请求测试老师",
                "campus": "红旗校区",
                "course_name": "重复请求测试课程",
                "content": "测试重复请求",
                "attitude": 1
            }
            
            # 第一次请求
            resp1 = self.client.post(
                self._url("/reviews/"),
                headers=headers,
                json=request_data
            )
            
            # 第二次请求（使用相同的幂等性Key）
            resp2 = self.client.post(
                self._url("/reviews/"),
                headers=headers,
                json=request_data
            )
            
            # 两次请求都应该成功，且第二次应该有幂等性重放标记
            passed = (
                resp1.status_code == 200 and
                resp2.status_code == 200 and
                resp2.headers.get("X-Idempotency-Replayed") == "true"
            )
            
            message = f"first={resp1.status_code}, second={resp2.status_code}, replayed={resp2.headers.get('X-Idempotency-Replayed')}"
            self._record("幂等性-重复请求", passed, message)
            return passed
        except Exception as e:
            self._record("幂等性-重复请求", False, str(e))
            return False

    def test_idempotency_without_key(self) -> bool:
        """测试幂等性：没有幂等性Key的请求（宽松模式应继续处理）"""
        try:
            # 不添加 X-Idempotency-Key 头部
            resp = self.client.post(
                self._url("/reviews/"),
                headers=self._headers(),
                json={
                    "teacher_name": "无Key测试老师",
                    "campus": "红旗校区",
                    "course_name": "无Key测试课程",
                    "content": "测试无幂等性Key",
                    "attitude": 1
                }
            )
            # 宽松模式下应该仍然处理请求
            passed = resp.status_code == 200
            self._record("幂等性-无Key请求", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("幂等性-无Key请求", False, str(e))
            return False

    def run_all_tests(self):
        """运行所有测试"""
        print("=" * 60)
        print("GoJxust API E2E 测试")
        print(f"Base URL: {self.base_url}")
        print("=" * 60)

        # 健康检查
        print("\n📋 基础测试")
        print("-" * 40)
        if not self.test_health_check():
            print("❌ 健康检查失败，服务可能未启动")
            return

        # 认证
        print("\n🔐 认证测试")
        print("-" * 40)
        if not self.setup_auth():
            print("❌ 认证失败，无法继续测试需要认证的接口")
            # 仍然继续测试公开接口

        # 公开接口
        print("\n🌐 公开接口测试")
        print("-" * 40)
        self.test_get_reviews_by_teacher()
        self.test_get_config_by_key()
        self.test_list_heroes()
        self.test_get_notifications()
        self.test_get_categories()

        if self.token:
            # 用户接口
            print("\n👤 用户接口测试")
            print("-" * 40)
            self.test_get_profile()
            self.test_update_profile()

            # 评价接口
            print("\n📝 评价接口测试")
            print("-" * 40)
            self.test_create_review()
            self.test_get_user_reviews()

            # 课程表接口
            print("\n📅 课程表接口测试")
            print("-" * 40)
            self.test_get_course_table()
            self.test_search_classes()

            # 挂科率接口
            print("\n📊 挂科率接口测试")
            print("-" * 40)
            self.test_search_fail_rate()
            self.test_rand_fail_rate()

            # 积分接口
            print("\n💰 积分接口测试")
            print("-" * 40)
            self.test_get_user_points()
            self.test_get_points_transactions()
            self.test_get_points_stats()

            # 投稿接口
            print("\n📤 投稿接口测试")
            print("-" * 40)
            self.test_create_contribution()
            self.test_get_contributions()
            self.test_get_user_contribution_stats()

            # 倒数日接口
            print("\n⏰ 倒数日接口测试")
            print("-" * 40)
            countdown_id = self.test_create_countdown()
            self.test_get_countdowns()
            if countdown_id:
                self.test_update_countdown(countdown_id)
                self.test_delete_countdown(countdown_id)

            # 学习任务接口
            print("\n📚 学习任务接口测试")
            print("-" * 40)
            task_id = self.test_create_study_task()
            self.test_get_study_tasks()
            self.test_get_study_task_stats()
            self.test_get_completed_tasks()
            if task_id:
                self.test_update_study_task(task_id)
                self.test_delete_study_task(task_id)

            # 幂等性接口测试
            print("\n🔁 幂等性接口测试")
            print("-" * 40)
            self.test_idempotency_create_review()
            self.test_idempotency_duplicate_request()
            self.test_idempotency_without_key()

        if self.admin_token:
            # 管理员接口
            print("\n🔧 管理员接口测试")
            print("-" * 40)
            self.test_admin_get_reviews()
            self.test_admin_get_notifications()
            self.test_admin_get_notification_stats()
            self.test_admin_search_heroes()
            self.test_admin_search_configs()

            # 功能白名单接口
            print("\n🎯 功能白名单接口测试")
            print("-" * 40)
            feature_key = self.test_admin_create_feature()
            self.test_admin_list_features()
            if feature_key:
                self.test_admin_update_feature(feature_key)
                self.test_admin_grant_feature(feature_key, user_id=1)
                self.test_admin_list_whitelist(feature_key)
                self.test_admin_get_user_features(user_id=1)
                self.test_admin_revoke_feature(feature_key, user_id=1)
                self.test_admin_delete_feature(feature_key)
            
            # 用户查看自己的功能列表
            if self.token:
                print("\n👤 用户功能列表测试")
                print("-" * 40)
                self.test_get_user_features()

        # 打印总结
        print("\n" + "=" * 60)
        print("测试总结")
        print("=" * 60)
        total = len(self.results)
        passed = sum(1 for r in self.results if r.passed)
        failed = total - passed
        print(f"总计: {total} | 通过: {passed} | 失败: {failed}")
        
        if failed > 0:
            print("\n失败的测试:")
            for r in self.results:
                if not r.passed:
                    print(f"  - {r.name}: {r.message}")
        
        print("=" * 60)
        return failed == 0

    def close(self):
        """关闭客户端"""
        self.client.close()


def main():
    parser = argparse.ArgumentParser(description="GoJxust API E2E 测试")
    parser.add_argument(
        "--base-url",
        default=DEFAULT_BASE_URL,
        help=f"API 基础 URL (默认: {DEFAULT_BASE_URL})"
    )
    parser.add_argument(
        "--insecure",
        action="store_true",
        help="允许不安全的 HTTPS 连接（忽略证书错误）"
    )
    args = parser.parse_args()

    client = E2ETestClient(args.base_url, insecure=args.insecure)
    try:
        success = client.run_all_tests()
        sys.exit(0 if success else 1)
    finally:
        client.close()


if __name__ == "__main__":
    main()
