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
from datetime import date, datetime, timedelta
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
        self.operator_token: Optional[str] = None
        self.refresh_token: Optional[str] = None
        self.admin_refresh_token: Optional[str] = None
        self.operator_refresh_token: Optional[str] = None
        self.admin_password_login_token: Optional[str] = None
        self.operator_password_login_token: Optional[str] = None
        self.basic_user_id: Optional[int] = None
        self.admin_user_id: Optional[int] = None
        self.operator_user_id: Optional[int] = None
        self.notification_category_id: Optional[int] = None
        self.results: list[TestResult] = []

    def _url(self, path: str) -> str:
        """构建完整 URL"""
        return f"{self.base_url}{API_PREFIX}{path}"

    def _headers(self, use_admin: bool = False, token: Optional[str] = None) -> dict:
        """获取请求头"""
        auth_token = token if token is not None else (self.admin_token if use_admin else self.token)
        if auth_token:
            return {"Authorization": f"Bearer {auth_token}"}
        return {}

    def _record(self, name: str, passed: bool, message: str):
        """记录测试结果"""
        status = "✅ PASS" if passed else "❌ FAIL"
        print(f"{status}: {name} - {message}")
        self.results.append(TestResult(name, passed, message))

    def _extract_result(self, resp: httpx.Response) -> dict:
        """提取统一信封中的 Result 字段"""
        return resp.json().get("Result", {})

    def _extract_page_data(self, resp: httpx.Response) -> list[dict]:
        """提取分页响应中的 data 列表"""
        result = self._extract_result(resp)
        if isinstance(result, dict):
            data = result.get("data", [])
            if isinstance(data, list):
                return data
        return []

    def _find_first(self, items: list[dict], **matches) -> Optional[dict]:
        """按键值匹配查找第一条记录"""
        for item in items:
            if not isinstance(item, dict):
                continue
            if all(item.get(key) == value for key, value in matches.items()):
                return item
        return None

    def _post_json(
        self,
        path: str,
        *,
        json: Optional[dict] = None,
        use_admin: bool = False,
        token: Optional[str] = None,
    ) -> httpx.Response:
        return self.client.post(
            self._url(path),
            headers=self._headers(use_admin=use_admin, token=token),
            json=json,
        )

    def _get(
        self,
        path: str,
        *,
        params: Optional[dict] = None,
        use_admin: bool = False,
        token: Optional[str] = None,
    ) -> httpx.Response:
        return self.client.get(
            self._url(path),
            headers=self._headers(use_admin=use_admin, token=token),
            params=params,
        )

    def _put_json(
        self,
        path: str,
        *,
        json: Optional[dict] = None,
        use_admin: bool = False,
        token: Optional[str] = None,
    ) -> httpx.Response:
        return self.client.put(
            self._url(path),
            headers=self._headers(use_admin=use_admin, token=token),
            json=json,
        )

    def _delete(
        self,
        path: str,
        *,
        use_admin: bool = False,
        token: Optional[str] = None,
    ) -> httpx.Response:
        return self.client.delete(
            self._url(path),
            headers=self._headers(use_admin=use_admin, token=token),
        )

    def _fetch_profile(self, token: str, label: str) -> Optional[dict]:
        """根据指定 token 获取当前用户资料"""
        try:
            resp = self.client.get(
                self._url("/user/profile"),
                headers=self._headers(token=token),
            )
            if resp.status_code != 200:
                self._record(f"读取{label}资料", False, f"status={resp.status_code}, body={resp.text}")
                return None
            result = self._extract_result(resp)
            self._record(f"读取{label}资料", True, f"user_id={result.get('id')}")
            return result
        except Exception as e:
            self._record(f"读取{label}资料", False, str(e))
            return None

    def _future_date(self, days: int = 30) -> str:
        """生成未来日期（YYYY-MM-DD）"""
        return (date.today() + timedelta(days=days)).isoformat()

    def _future_datetime(self, days: int = 30) -> str:
        """生成未来时间（YYYY-MM-DD HH:MM）"""
        future = datetime.now() + timedelta(days=days)
        return future.strftime("%Y-%m-%d %H:%M")

    def _fetch_notification_categories(self, token: str) -> list[dict]:
        """获取通知分类列表"""
        try:
            resp = self.client.get(
                self._url("/categories/"),
                headers=self._headers(token=token),
            )
            if resp.status_code != 200:
                return []
            result = self._extract_result(resp)
            return result if isinstance(result, list) else []
        except Exception:
            return []

    def ensure_notification_category(self) -> Optional[int]:
        """确保存在一个可用于测试的通知分类"""
        categories: list[dict] = []
        if self.token:
            categories = self._fetch_notification_categories(self.token)
        if not categories and self.admin_token:
            categories = self._fetch_notification_categories(self.admin_token)

        if categories:
            category_id = categories[0].get("id")
            if category_id:
                self.notification_category_id = int(category_id)
                self._record("加载通知分类", True, f"id={category_id}")
                return self.notification_category_id

        if not self.admin_token:
            self._record("加载通知分类", False, "无管理员 token，无法自动创建分类")
            return None

        try:
            resp = self.client.post(
                self._url("/admin/categories/"),
                headers=self._headers(token=self.admin_token),
                json={
                    "name": f"E2E分类{uuid.uuid4().hex[:6]}",
                    "sort": 999,
                    "is_active": True,
                },
            )
            if resp.status_code != 200:
                self._record("创建测试分类", False, f"status={resp.status_code}, body={resp.text}")
                return None

            result = self._extract_result(resp)
            category_id = result.get("id")
            if not category_id:
                self._record("创建测试分类", False, f"响应缺少分类ID, body={resp.text}")
                return None

            self.notification_category_id = int(category_id)
            self._record("创建测试分类", True, f"id={category_id}")
            return self.notification_category_id
        except Exception as e:
            self._record("创建测试分类", False, str(e))
            return None

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

    def test_mock_wechat_login(self, test_user: str = "basic") -> Optional[dict]:
        """测试模拟微信登录"""
        try:
            resp = self._post_json("/auth/mock-wechat-login", json={"test_user": test_user})
            if resp.status_code == 200:
                result = self._extract_result(resp)
                if result.get("token"):
                    self._record(f"模拟登录({test_user})", True, "获取 token 成功")
                    return result
                self._record(f"模拟登录({test_user})", False, f"token 为空, body={resp.text}")
                return None
            self._record(f"模拟登录({test_user})", False, f"status={resp.status_code}, body={resp.text}")
            return None
        except Exception as e:
            self._record(f"模拟登录({test_user})", False, str(e))
            return None

    def setup_auth(self) -> bool:
        """设置认证 token"""
        basic_login = self.test_mock_wechat_login("basic")
        admin_login = self.test_mock_wechat_login("admin")
        operator_login = self.test_mock_wechat_login("operator")

        if basic_login:
            self.token = basic_login.get("token")
            self.refresh_token = basic_login.get("refresh_token")
        if admin_login:
            self.admin_token = admin_login.get("token")
            self.admin_refresh_token = admin_login.get("refresh_token")
        if operator_login:
            self.operator_token = operator_login.get("token")
            self.operator_refresh_token = operator_login.get("refresh_token")

        if self.token:
            profile = self._fetch_profile(self.token, "basic用户")
            if profile:
                self.basic_user_id = profile.get("id")
        if self.admin_token:
            profile = self._fetch_profile(self.admin_token, "admin用户")
            if profile:
                self.admin_user_id = profile.get("id")
        if self.operator_token:
            profile = self._fetch_profile(self.operator_token, "operator用户")
            if profile:
                self.operator_user_id = profile.get("id")
        if self.token or self.admin_token:
            self.ensure_notification_category()

        return self.token is not None

    def test_admin_set_login_credentials(
        self,
        name: str,
        caller_token: str,
        target_user_id: int,
        phone: str,
        password: str,
        expected_status: int = 200,
    ) -> bool:
        """测试管理员设置后台登录凭据"""
        try:
            resp = self.client.put(
                self._url(f"/admin/users/{target_user_id}/login-credentials"),
                headers=self._headers(token=caller_token),
                json={"phone": phone, "password": password},
            )
            passed = resp.status_code == expected_status
            self._record(name, passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record(name, False, str(e))
            return False

    def test_admin_password_login(
        self,
        name: str,
        phone: str,
        password: str,
        expected_status: int = 200,
    ) -> Optional[str]:
        """测试后台手机号密码登录"""
        try:
            resp = self.client.post(
                self._url("/admin/auth/login"),
                json={"phone": phone, "password": password},
            )
            passed = resp.status_code == expected_status
            token = None
            if passed and expected_status == 200:
                token = self._extract_result(resp).get("token")
                passed = token is not None
            self._record(name, passed, f"status={resp.status_code}")
            return token if passed and expected_status == 200 else None
        except Exception as e:
            self._record(name, False, str(e))
            return None

    def test_admin_notifications_with_token(self, name: str, token: str) -> bool:
        """测试指定后台 token 访问通知后台列表"""
        try:
            resp = self.client.get(
                self._url("/admin/notifications/"),
                headers=self._headers(token=token),
                params={"page": 1, "size": 10},
            )
            passed = resp.status_code == 200
            self._record(name, passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record(name, False, str(e))
            return False

    def test_admin_user_detail_with_token(self, name: str, token: str, user_id: int) -> bool:
        """测试指定后台 token 访问用户管理详情"""
        try:
            resp = self.client.get(
                self._url(f"/admin/users/{user_id}"),
                headers=self._headers(token=token),
            )
            passed = resp.status_code == 200
            self._record(name, passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record(name, False, str(e))
            return False

    def test_refresh_token(self) -> bool:
        """测试刷新访问令牌"""
        if not self.refresh_token:
            self._record("刷新访问令牌", False, "缺少 refresh token")
            return False
        try:
            resp = self._post_json("/auth/refresh", json={"refresh_token": self.refresh_token})
            passed = resp.status_code == 200 and bool(self._extract_result(resp).get("token"))
            self._record("刷新访问令牌", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("刷新访问令牌", False, str(e))
            return False

    def test_logout_temp_user(self) -> bool:
        """测试退出当前设备登录"""
        login_result = self.test_mock_wechat_login("active")
        if not login_result or not login_result.get("token"):
            self._record("退出当前设备登录", False, "无法创建临时 active 会话")
            return False
        temp_token = login_result["token"]
        try:
            resp = self._post_json("/auth/logout", token=temp_token)
            if resp.status_code != 200:
                self._record("退出当前设备登录", False, f"status={resp.status_code}")
                return False
            verify_resp = self._get("/user/profile", token=temp_token)
            passed = verify_resp.status_code != 200
            self._record("退出当前设备登录", passed, f"logout=200, profile_after={verify_resp.status_code}")
            return passed
        except Exception as e:
            self._record("退出当前设备登录", False, str(e))
            return False

    def test_logout_all_temp_user(self) -> bool:
        """测试退出全部设备登录"""
        login1 = self.test_mock_wechat_login("verified")
        login2 = self.test_mock_wechat_login("verified")
        if not login1 or not login2:
            self._record("退出全部设备登录", False, "无法创建 verified 多会话")
            return False
        token1 = login1.get("token")
        token2 = login2.get("token")
        if not token1 or not token2:
            self._record("退出全部设备登录", False, "verified token 为空")
            return False
        try:
            resp = self._post_json("/auth/logout-all", token=token1)
            if resp.status_code != 200:
                self._record("退出全部设备登录", False, f"status={resp.status_code}")
                return False
            verify1 = self._get("/user/profile", token=token1)
            verify2 = self._get("/user/profile", token=token2)
            passed = verify1.status_code != 200 and verify2.status_code != 200
            self._record("退出全部设备登录", passed, f"after=({verify1.status_code},{verify2.status_code})")
            return passed
        except Exception as e:
            self._record("退出全部设备登录", False, str(e))
            return False

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

    def test_get_login_days(self) -> bool:
        """测试获取登录天数"""
        try:
            resp = self._get("/user/login-days")
            passed = resp.status_code == 200
            self._record("获取登录天数", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取登录天数", False, str(e))
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
        """测试获取通知列表（需认证）"""
        try:
            resp = self.client.get(
                self._url("/notifications/"),
                headers=self._headers(),
                params={"page": 1, "size": 10}
            )
            passed = resp.status_code == 200
            self._record("获取通知列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取通知列表", False, str(e))
            return False

    def test_get_categories(self) -> bool:
        """测试获取分类列表（需认证）"""
        try:
            resp = self.client.get(
                self._url("/categories/"),
                headers=self._headers()
            )
            passed = resp.status_code == 200
            self._record("获取分类列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取分类列表", False, str(e))
            return False

    def test_get_notification_detail(self) -> bool:
        """测试获取通知详情（如有数据）"""
        try:
            list_resp = self._get("/notifications/", params={"page": 1, "size": 10})
            if list_resp.status_code != 200:
                self._record("获取通知详情", False, f"列表 status={list_resp.status_code}")
                return False
            items = self._extract_page_data(list_resp)
            if not items:
                self._record("获取通知详情", True, "列表为空，跳过详情验证")
                return True
            notification_id = items[0].get("id")
            resp = self._get(f"/notifications/{notification_id}")
            passed = resp.status_code == 200
            self._record("获取通知详情", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("获取通知详情", False, str(e))
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

    def test_get_course_table_bind_count(self) -> bool:
        """测试获取课程表绑定次数"""
        try:
            resp = self._get("/coursetable/bind-count")
            passed = resp.status_code == 200
            self._record("获取课表绑定次数", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取课表绑定次数", False, str(e))
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

    def test_create_contribution(self) -> Optional[str]:
        """测试创建投稿"""
        if not self.notification_category_id:
            self._record("创建投稿", False, "缺少可用通知分类ID")
            return None
        title = f"E2E测试投稿-{uuid.uuid4().hex[:8]}"
        try:
            resp = self._post_json(
                "/contributions/",
                json={
                    "title": title,
                    "content": "这是E2E测试创建的投稿内容",
                    "categories": [self.notification_category_id]
                },
            )
            passed = resp.status_code == 200
            self._record("创建投稿", passed, f"status={resp.status_code}")
            return title if passed else None
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

    def test_get_contribution_detail(self, title: str) -> Optional[int]:
        """测试获取投稿详情"""
        try:
            resp = self._get("/contributions/", params={"page": 1, "size": 20})
            if resp.status_code != 200:
                self._record("获取投稿详情", False, f"list_status={resp.status_code}")
                return None
            items = self._extract_page_data(resp)
            item = self._find_first(items, title=title)
            if not item:
                self._record("获取投稿详情", False, f"未找到标题={title}")
                return None
            contribution_id = item.get("id")
            detail_resp = self._get(f"/contributions/{contribution_id}")
            passed = detail_resp.status_code == 200
            self._record("获取投稿详情", passed, f"status={detail_resp.status_code}, id={contribution_id}")
            return int(contribution_id) if passed and contribution_id else None
        except Exception as e:
            self._record("获取投稿详情", False, str(e))
            return None

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
                    "target_date": self._future_date()
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

    def test_get_countdown_detail(self, countdown_id: int) -> bool:
        """测试获取倒数日详情"""
        try:
            resp = self._get(f"/countdowns/{countdown_id}")
            passed = resp.status_code == 200
            self._record("获取倒数日详情", passed, f"status={resp.status_code}, id={countdown_id}")
            return passed
        except Exception as e:
            self._record("获取倒数日详情", False, str(e))
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
                    "due_date": self._future_datetime(),
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

    def test_get_study_task_detail(self, task_id: int) -> bool:
        """测试获取学习任务详情"""
        try:
            resp = self._get(f"/study-tasks/{task_id}")
            passed = resp.status_code == 200
            self._record("获取学习任务详情", passed, f"status={resp.status_code}, id={task_id}")
            return passed
        except Exception as e:
            self._record("获取学习任务详情", False, str(e))
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

    # ==================== 资料/刷题/统计等（需认证）====================

    def test_get_material_categories(self) -> bool:
        try:
            resp = self._get("/material-categories/")
            passed = resp.status_code == 200
            self._record("获取资料分类", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取资料分类", False, str(e))
            return False

    def test_get_materials(self) -> Optional[str]:
        try:
            resp = self._get("/materials/", params={"page": 1, "page_size": 10})
            if resp.status_code != 200:
                self._record("获取资料列表", False, f"status={resp.status_code}")
                return None
            items = self._extract_page_data(resp)
            md5 = items[0].get("md5") if items else None
            self._record("获取资料列表", True, f"count={len(items)}")
            return md5
        except Exception as e:
            self._record("获取资料列表", False, str(e))
            return None

    def test_get_top_materials(self) -> bool:
        try:
            resp = self._get("/materials/top", params={"limit": 5})
            passed = resp.status_code == 200
            self._record("获取热门资料", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取热门资料", False, str(e))
            return False

    def test_get_material_hot_words(self) -> bool:
        try:
            resp = self._get("/materials/hot-words", params={"limit": 5})
            passed = resp.status_code == 200
            self._record("获取资料热词", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取资料热词", False, str(e))
            return False

    def test_search_materials(self) -> bool:
        try:
            resp = self._get("/materials/search", params={"keywords": "考试", "page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("搜索资料", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("搜索资料", False, str(e))
            return False

    def test_get_material_detail(self, md5: str) -> bool:
        try:
            resp = self._get(f"/materials/{md5}")
            passed = resp.status_code == 200
            self._record("获取资料详情", passed, f"status={resp.status_code}, md5={md5}")
            return passed
        except Exception as e:
            self._record("获取资料详情", False, str(e))
            return False

    def test_rate_material(self, md5: str) -> bool:
        try:
            resp = self._post_json(f"/materials/{md5}/rating", json={"rating": 5})
            passed = resp.status_code == 200
            self._record("资料评分", passed, f"status={resp.status_code}, md5={md5}")
            return passed
        except Exception as e:
            self._record("资料评分", False, str(e))
            return False

    def test_download_material(self, md5: str) -> bool:
        try:
            resp = self._post_json(f"/materials/{md5}/download", json={})
            passed = resp.status_code == 200
            self._record("记录资料下载", passed, f"status={resp.status_code}, md5={md5}")
            return passed
        except Exception as e:
            self._record("记录资料下载", False, str(e))
            return False

    def test_get_question_projects(self) -> Optional[int]:
        try:
            resp = self._get("/questions/projects")
            if resp.status_code != 200:
                self._record("获取刷题项目", False, f"status={resp.status_code}")
                return None
            result = self._extract_result(resp)
            projects = result if isinstance(result, list) else []
            project_id = projects[0].get("id") if projects else None
            self._record("获取刷题项目", True, f"count={len(projects)}")
            return int(project_id) if project_id else None
        except Exception as e:
            self._record("获取刷题项目", False, str(e))
            return None

    def test_get_question_list(self, project_id: int) -> Optional[int]:
        try:
            resp = self._get("/questions/list", params={"project_id": project_id})
            if resp.status_code != 200:
                self._record("获取题目ID列表", False, f"status={resp.status_code}, project_id={project_id}")
                return None
            question_ids = self._extract_result(resp).get("question_ids", [])
            first_question_id = question_ids[0] if question_ids else None
            self._record("获取题目ID列表", True, f"count={len(question_ids)}, project_id={project_id}")
            return int(first_question_id) if first_question_id else None
        except Exception as e:
            self._record("获取题目ID列表", False, str(e))
            return None

    def test_get_question_detail(self, question_id: int) -> bool:
        try:
            resp = self._get(f"/questions/{question_id}")
            passed = resp.status_code == 200
            self._record("获取题目详情", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("获取题目详情", False, str(e))
            return False

    def test_record_question_study(self, question_id: int, project_id: int) -> bool:
        try:
            resp = self._post_json("/questions/study", json={"question_id": question_id, "project_id": project_id})
            passed = resp.status_code == 200
            self._record("记录题目学习", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("记录题目学习", False, str(e))
            return False

    def test_submit_question_practice(self, question_id: int, project_id: int) -> bool:
        try:
            resp = self._post_json("/questions/practice", json={"question_id": question_id, "project_id": project_id})
            passed = resp.status_code == 200
            self._record("记录题目练习", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("记录题目练习", False, str(e))
            return False

    def test_get_pomodoro_count(self) -> bool:
        try:
            resp = self._get("/pomodoro/count")
            passed = resp.status_code == 200
            self._record("获取番茄钟次数", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取番茄钟次数", False, str(e))
            return False

    def test_increment_pomodoro(self) -> bool:
        try:
            resp = self._post_json("/pomodoro/increment", json={})
            passed = resp.status_code == 200
            self._record("增加番茄钟次数", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("增加番茄钟次数", False, str(e))
            return False

    def test_get_pomodoro_ranking(self) -> bool:
        try:
            resp = self._get("/pomodoro/ranking")
            passed = resp.status_code == 200
            self._record("获取番茄钟排名", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取番茄钟排名", False, str(e))
            return False

    def test_get_system_online(self) -> bool:
        try:
            resp = self._get("/stat/system/online")
            passed = resp.status_code == 200
            self._record("获取系统在线人数", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取系统在线人数", False, str(e))
            return False

    def test_get_project_online(self, project_id: int) -> bool:
        try:
            resp = self._get(f"/stat/project/{project_id}/online")
            passed = resp.status_code == 200
            self._record("获取项目在线人数", passed, f"status={resp.status_code}, project_id={project_id}")
            return passed
        except Exception as e:
            self._record("获取项目在线人数", False, str(e))
            return False

    def test_create_gpa_backup(self) -> Optional[int]:
        try:
            payload = {
                "semester": "2024-2025-1",
                "courses": [
                    {"name": "高等数学", "credit": 4, "score": 91},
                    {"name": "大学英语", "credit": 2, "score": 88},
                ],
            }
            resp = self._post_json("/gpa/backup", json=payload)
            if resp.status_code != 200:
                self._record("创建绩点备份", False, f"status={resp.status_code}")
                return None
            backup_id = self._extract_result(resp).get("id")
            self._record("创建绩点备份", bool(backup_id), f"status={resp.status_code}, id={backup_id}")
            return int(backup_id) if backup_id else None
        except Exception as e:
            self._record("创建绩点备份", False, str(e))
            return None

    def test_list_gpa_backups(self) -> bool:
        try:
            resp = self._get("/gpa/backup")
            passed = resp.status_code == 200
            self._record("获取绩点备份列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取绩点备份列表", False, str(e))
            return False

    def test_get_gpa_backup_detail(self, backup_id: int) -> bool:
        try:
            resp = self._get(f"/gpa/backup/{backup_id}")
            passed = resp.status_code == 200
            self._record("获取绩点备份详情", passed, f"status={resp.status_code}, id={backup_id}")
            return passed
        except Exception as e:
            self._record("获取绩点备份详情", False, str(e))
            return False

    def test_delete_gpa_backup(self, backup_id: int) -> bool:
        try:
            resp = self._delete(f"/gpa/backup/{backup_id}")
            passed = resp.status_code == 200
            self._record("删除绩点备份", passed, f"status={resp.status_code}, id={backup_id}")
            return passed
        except Exception as e:
            self._record("删除绩点备份", False, str(e))
            return False

    def test_get_dictionary_word(self) -> bool:
        try:
            resp = self._get("/dictionary/word")
            passed = resp.status_code == 200
            self._record("获取随机词典条目", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("获取随机词典条目", False, str(e))
            return False

    # ==================== 管理链路扩展 ====================

    def test_admin_grant_points(self, user_id: int, points: int = 5) -> bool:
        try:
            resp = self._post_json(
                "/points/grant",
                use_admin=True,
                json={"user_id": user_id, "points": points, "description": "E2E管理员发放积分"},
            )
            passed = resp.status_code == 200
            self._record("管理员发放积分", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员发放积分", False, str(e))
            return False

    def test_spend_points(self) -> bool:
        try:
            resp = self._post_json("/points/spend", json={"points": 1, "description": "E2E消费积分"})
            passed = resp.status_code == 200
            self._record("用户消费积分", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("用户消费积分", False, str(e))
            return False

    def test_admin_get_user_points(self, user_id: int) -> bool:
        try:
            resp = self._get("/points/", use_admin=True, params={"user_id": user_id})
            passed = resp.status_code == 200
            self._record("管理员获取指定用户积分", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员获取指定用户积分", False, str(e))
            return False

    def test_admin_get_user_point_stats(self, user_id: int) -> bool:
        try:
            resp = self._get("/points/stats", use_admin=True, params={"user_id": user_id})
            passed = resp.status_code == 200
            self._record("管理员获取指定用户积分统计", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员获取指定用户积分统计", False, str(e))
            return False

    def test_admin_get_user_point_transactions(self, user_id: int) -> bool:
        try:
            resp = self._get("/points/transactions", use_admin=True, params={"page": 1, "size": 10, "user_id": user_id})
            passed = resp.status_code == 200
            self._record("管理员获取指定用户积分流水", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员获取指定用户积分流水", False, str(e))
            return False

    def test_admin_create_hero(self) -> Optional[int]:
        hero_name = f"E2E英雄-{uuid.uuid4().hex[:6]}"
        try:
            resp = self._post_json("/heroes/", use_admin=True, json={"name": hero_name, "sort": 999, "is_show": True})
            if resp.status_code != 200:
                self._record("管理员创建英雄", False, f"status={resp.status_code}")
                return None
            hero_id = self._extract_result(resp).get("id")
            self._record("管理员创建英雄", bool(hero_id), f"status={resp.status_code}, id={hero_id}")
            return int(hero_id) if hero_id else None
        except Exception as e:
            self._record("管理员创建英雄", False, str(e))
            return None

    def test_admin_update_hero(self, hero_id: int) -> bool:
        try:
            resp = self._put_json(f"/heroes/{hero_id}", use_admin=True, json={"name": "E2E英雄-已更新", "sort": 1000, "is_show": True})
            passed = resp.status_code == 200
            self._record("管理员更新英雄", passed, f"status={resp.status_code}, id={hero_id}")
            return passed
        except Exception as e:
            self._record("管理员更新英雄", False, str(e))
            return False

    def test_admin_delete_hero(self, hero_id: int) -> bool:
        try:
            resp = self._delete(f"/heroes/{hero_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除英雄", passed, f"status={resp.status_code}, id={hero_id}")
            return passed
        except Exception as e:
            self._record("管理员删除英雄", False, str(e))
            return False

    def test_admin_create_config(self) -> Optional[str]:
        config_key = f"e2e.config.{uuid.uuid4().hex[:8]}"
        try:
            resp = self._post_json(
                "/config/",
                use_admin=True,
                json={"key": config_key, "value": "true", "value_type": "boolean", "description": "E2E配置"},
            )
            passed = resp.status_code == 200
            self._record("管理员创建配置", passed, f"status={resp.status_code}, key={config_key}")
            return config_key if passed else None
        except Exception as e:
            self._record("管理员创建配置", False, str(e))
            return None

    def test_admin_update_config(self, config_key: str) -> bool:
        try:
            resp = self._put_json(
                f"/config/{config_key}",
                use_admin=True,
                json={"value": '{"enabled":true}', "value_type": "json", "description": "E2E配置已更新"},
            )
            passed = resp.status_code == 200
            self._record("管理员更新配置", passed, f"status={resp.status_code}, key={config_key}")
            return passed
        except Exception as e:
            self._record("管理员更新配置", False, str(e))
            return False

    def test_get_config_by_specific_key(self, config_key: str) -> bool:
        try:
            resp = self.client.get(self._url(f"/config/{config_key}"))
            passed = resp.status_code == 200
            self._record("读取新建配置", passed, f"status={resp.status_code}, key={config_key}")
            return passed
        except Exception as e:
            self._record("读取新建配置", False, str(e))
            return False

    def test_admin_delete_config(self, config_key: str) -> bool:
        try:
            resp = self._delete(f"/config/{config_key}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除配置", passed, f"status={resp.status_code}, key={config_key}")
            return passed
        except Exception as e:
            self._record("管理员删除配置", False, str(e))
            return False

    def test_admin_update_category(self, category_id: int) -> bool:
        try:
            resp = self._put_json(f"/admin/categories/{category_id}", use_admin=True, json={"name": f"E2E分类已更新{category_id}", "sort": 998})
            passed = resp.status_code == 200
            self._record("管理员更新通知分类", passed, f"status={resp.status_code}, id={category_id}")
            return passed
        except Exception as e:
            self._record("管理员更新通知分类", False, str(e))
            return False

    def test_operator_create_notification(self) -> Optional[int]:
        if not self.operator_token or not self.notification_category_id:
            self._record("运营创建通知", False, "缺少 operator token 或 category id")
            return None
        try:
            resp = self._post_json(
                "/admin/notifications/",
                token=self.operator_token,
                json={
                    "title": f"E2E通知-{uuid.uuid4().hex[:8]}",
                    "content": "E2E通知内容",
                    "categories": [self.notification_category_id],
                },
            )
            if resp.status_code != 200:
                self._record("运营创建通知", False, f"status={resp.status_code}, body={resp.text}")
                return None
            notification_id = self._extract_result(resp).get("id")
            self._record("运营创建通知", bool(notification_id), f"status={resp.status_code}, id={notification_id}")
            return int(notification_id) if notification_id else None
        except Exception as e:
            self._record("运营创建通知", False, str(e))
            return None

    def test_admin_get_notification_detail(self, notification_id: int) -> bool:
        try:
            resp = self._get(f"/admin/notifications/{notification_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取通知详情", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员获取通知详情", False, str(e))
            return False

    def test_admin_update_notification(self, notification_id: int) -> bool:
        try:
            resp = self._put_json(
                f"/admin/notifications/{notification_id}",
                use_admin=True,
                json={"title": "E2E通知-已更新", "content": "E2E通知内容已更新", "categories": [self.notification_category_id]},
            )
            passed = resp.status_code == 200
            self._record("管理员更新通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员更新通知", False, str(e))
            return False

    def test_admin_approve_notification(self, notification_id: int) -> bool:
        try:
            resp = self._post_json(f"/admin/notifications/{notification_id}/approve", use_admin=True, json={"status": 1, "note": "E2E审核通过"})
            passed = resp.status_code == 200
            self._record("管理员审核通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员审核通知", False, str(e))
            return False

    def test_admin_publish_notification(self, notification_id: int) -> bool:
        try:
            resp = self._post_json(f"/admin/notifications/{notification_id}/publish-admin", use_admin=True, json={})
            passed = resp.status_code == 200
            self._record("管理员直接发布通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员直接发布通知", False, str(e))
            return False

    def test_admin_pin_notification(self, notification_id: int) -> bool:
        try:
            resp = self._post_json(f"/admin/notifications/{notification_id}/pin", use_admin=True, json={})
            passed = resp.status_code == 200
            self._record("管理员置顶通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员置顶通知", False, str(e))
            return False

    def test_admin_unpin_notification(self, notification_id: int) -> bool:
        try:
            resp = self._post_json(f"/admin/notifications/{notification_id}/unpin", use_admin=True, json={})
            passed = resp.status_code == 200
            self._record("管理员取消置顶通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员取消置顶通知", False, str(e))
            return False

    def test_admin_delete_notification(self, notification_id: int) -> bool:
        try:
            resp = self._delete(f"/admin/notifications/{notification_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除通知", passed, f"status={resp.status_code}, id={notification_id}")
            return passed
        except Exception as e:
            self._record("管理员删除通知", False, str(e))
            return False

    def test_admin_get_contribution_stats(self) -> bool:
        try:
            resp = self._get("/contributions/stats-admin", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取投稿统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取投稿统计", False, str(e))
            return False

    def test_admin_create_coursetable(self) -> Optional[int]:
        class_id = f"E2E-CLASS-{uuid.uuid4().hex[:8]}"
        try:
            resp = self._post_json(
                "/admin/coursetables",
                use_admin=True,
                json={
                    "class_id": class_id,
                    "semester": "2099-2100-1",
                    "course_data": {
                        "1": {
                            "name": "E2E课程",
                            "teacher": "E2E老师",
                            "location": "E2E教室",
                        }
                    },
                },
            )
            if resp.status_code != 200:
                self._record("管理员创建课表", False, f"status={resp.status_code}, body={resp.text}")
                return None
            coursetable_id = self._extract_result(resp).get("id")
            self._record("管理员创建课表", bool(coursetable_id), f"status={resp.status_code}, id={coursetable_id}")
            return int(coursetable_id) if coursetable_id else None
        except Exception as e:
            self._record("管理员创建课表", False, str(e))
            return None

    def test_admin_list_coursetables(self) -> bool:
        try:
            resp = self._get("/admin/coursetables", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取课表列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取课表列表", False, str(e))
            return False

    def test_admin_get_coursetable_detail(self, coursetable_id: int) -> bool:
        try:
            resp = self._get(f"/admin/coursetables/{coursetable_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取课表详情", passed, f"status={resp.status_code}, id={coursetable_id}")
            return passed
        except Exception as e:
            self._record("管理员获取课表详情", False, str(e))
            return False

    def test_admin_update_coursetable(self, coursetable_id: int) -> bool:
        try:
            resp = self._put_json(
                f"/admin/coursetables/{coursetable_id}",
                use_admin=True,
                json={
                    "course_data": {
                        "1": {
                            "name": "E2E课程-已更新",
                            "teacher": "E2E老师",
                            "location": "E2E教室A101",
                        }
                    }
                },
            )
            passed = resp.status_code == 200
            self._record("管理员更新课表", passed, f"status={resp.status_code}, id={coursetable_id}")
            return passed
        except Exception as e:
            self._record("管理员更新课表", False, str(e))
            return False

    def test_admin_delete_coursetable(self, coursetable_id: int) -> bool:
        try:
            resp = self._delete(f"/admin/coursetables/{coursetable_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除课表", passed, f"status={resp.status_code}, id={coursetable_id}")
            return passed
        except Exception as e:
            self._record("管理员删除课表", False, str(e))
            return False

    def test_admin_reset_coursetable_bind_count(self, user_id: int) -> bool:
        try:
            resp = self._post_json(f"/admin/users/{user_id}/coursetable-bind-count/reset", use_admin=True, json={})
            passed = resp.status_code == 200
            self._record("管理员重置课表绑定次数", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员重置课表绑定次数", False, str(e))
            return False

    def test_admin_create_failrate(self) -> Optional[int]:
        course_name = f"E2E挂科率课程-{uuid.uuid4().hex[:8]}"
        try:
            resp = self._post_json(
                "/admin/failrates",
                use_admin=True,
                json={
                    "course_name": course_name,
                    "department": "E2E学院",
                    "semester": "2099-2100-1",
                    "failrate": 12.5,
                },
            )
            if resp.status_code != 200:
                self._record("管理员创建挂科率", False, f"status={resp.status_code}, body={resp.text}")
                return None
            failrate_id = self._extract_result(resp).get("id")
            self._record("管理员创建挂科率", bool(failrate_id), f"status={resp.status_code}, id={failrate_id}")
            return int(failrate_id) if failrate_id else None
        except Exception as e:
            self._record("管理员创建挂科率", False, str(e))
            return None

    def test_admin_list_failrates(self) -> bool:
        try:
            resp = self._get("/admin/failrates", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取挂科率列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取挂科率列表", False, str(e))
            return False

    def test_admin_get_failrate_detail(self, failrate_id: int) -> bool:
        try:
            resp = self._get(f"/admin/failrates/{failrate_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取挂科率详情", passed, f"status={resp.status_code}, id={failrate_id}")
            return passed
        except Exception as e:
            self._record("管理员获取挂科率详情", False, str(e))
            return False

    def test_admin_update_failrate(self, failrate_id: int) -> bool:
        try:
            resp = self._put_json(
                f"/admin/failrates/{failrate_id}",
                use_admin=True,
                json={"failrate": 18.5, "department": "E2E学院-已更新"},
            )
            passed = resp.status_code == 200
            self._record("管理员更新挂科率", passed, f"status={resp.status_code}, id={failrate_id}")
            return passed
        except Exception as e:
            self._record("管理员更新挂科率", False, str(e))
            return False

    def test_admin_delete_failrate(self, failrate_id: int) -> bool:
        try:
            resp = self._delete(f"/admin/failrates/{failrate_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除挂科率", passed, f"status={resp.status_code}, id={failrate_id}")
            return passed
        except Exception as e:
            self._record("管理员删除挂科率", False, str(e))
            return False

    def test_admin_create_question_project(self) -> Optional[int]:
        try:
            resp = self._post_json(
                "/admin/questions/projects",
                use_admin=True,
                json={
                    "name": f"E2E题库项目-{uuid.uuid4().hex[:8]}",
                    "description": "E2E题库项目描述",
                    "version": 1,
                    "sort": 999,
                    "is_active": True,
                },
            )
            if resp.status_code != 200:
                self._record("管理员创建题库项目", False, f"status={resp.status_code}, body={resp.text}")
                return None
            project_id = self._extract_result(resp).get("id")
            self._record("管理员创建题库项目", bool(project_id), f"status={resp.status_code}, id={project_id}")
            return int(project_id) if project_id else None
        except Exception as e:
            self._record("管理员创建题库项目", False, str(e))
            return None

    def test_admin_list_question_projects(self) -> bool:
        try:
            resp = self._get("/admin/questions/projects", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取题库项目列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取题库项目列表", False, str(e))
            return False

    def test_admin_get_question_project_detail(self, project_id: int) -> bool:
        try:
            resp = self._get(f"/admin/questions/projects/{project_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取题库项目详情", passed, f"status={resp.status_code}, id={project_id}")
            return passed
        except Exception as e:
            self._record("管理员获取题库项目详情", False, str(e))
            return False

    def test_admin_update_question_project(self, project_id: int) -> bool:
        try:
            resp = self._put_json(
                f"/admin/questions/projects/{project_id}",
                use_admin=True,
                json={"description": "E2E题库项目描述-已更新", "sort": 1000},
            )
            passed = resp.status_code == 200
            self._record("管理员更新题库项目", passed, f"status={resp.status_code}, id={project_id}")
            return passed
        except Exception as e:
            self._record("管理员更新题库项目", False, str(e))
            return False

    def test_admin_delete_question_project(self, project_id: int) -> bool:
        try:
            resp = self._delete(f"/admin/questions/projects/{project_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除题库项目", passed, f"status={resp.status_code}, id={project_id}")
            return passed
        except Exception as e:
            self._record("管理员删除题库项目", False, str(e))
            return False

    def test_admin_create_question(self, project_id: int) -> Optional[int]:
        try:
            resp = self._post_json(
                "/admin/questions",
                use_admin=True,
                json={
                    "project_id": project_id,
                    "type": 1,
                    "title": f"E2E题目-{uuid.uuid4().hex[:8]}",
                    "options": ["选项A", "选项B", "选项C"],
                    "answer": "选项A",
                    "sort": 1,
                    "is_active": True,
                },
            )
            if resp.status_code != 200:
                self._record("管理员创建题目", False, f"status={resp.status_code}, body={resp.text}")
                return None
            question_id = self._extract_result(resp).get("id")
            self._record("管理员创建题目", bool(question_id), f"status={resp.status_code}, id={question_id}")
            return int(question_id) if question_id else None
        except Exception as e:
            self._record("管理员创建题目", False, str(e))
            return None

    def test_admin_list_questions(self, project_id: Optional[int] = None) -> bool:
        params = {"page": 1, "page_size": 10}
        if project_id:
            params["project_id"] = project_id
        try:
            resp = self._get("/admin/questions", use_admin=True, params=params)
            passed = resp.status_code == 200
            self._record("管理员获取题目列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取题目列表", False, str(e))
            return False

    def test_admin_get_question_detail(self, question_id: int) -> bool:
        try:
            resp = self._get(f"/admin/questions/{question_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取题目详情", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("管理员获取题目详情", False, str(e))
            return False

    def test_admin_update_question(self, question_id: int) -> bool:
        try:
            resp = self._put_json(
                f"/admin/questions/{question_id}",
                use_admin=True,
                json={"title": "E2E题目-已更新", "answer": "选项B", "sort": 2},
            )
            passed = resp.status_code == 200
            self._record("管理员更新题目", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("管理员更新题目", False, str(e))
            return False

    def test_admin_delete_question(self, question_id: int) -> bool:
        try:
            resp = self._delete(f"/admin/questions/{question_id}", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员删除题目", passed, f"status={resp.status_code}, id={question_id}")
            return passed
        except Exception as e:
            self._record("管理员删除题目", False, str(e))
            return False

    def test_admin_get_countdown_group_stats(self) -> bool:
        try:
            resp = self._get("/admin/stats/countdowns/by-user", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取倒数日分组统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取倒数日分组统计", False, str(e))
            return False

    def test_admin_get_studytask_group_stats(self) -> bool:
        try:
            resp = self._get("/admin/stats/studytasks/by-user", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取学习清单分组统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取学习清单分组统计", False, str(e))
            return False

    def test_admin_get_gpa_backup_group_stats(self) -> bool:
        try:
            resp = self._get("/admin/stats/gpa-backups/by-user", use_admin=True, params={"page": 1, "page_size": 10})
            passed = resp.status_code == 200
            self._record("管理员获取绩点备份分组统计", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取绩点备份分组统计", False, str(e))
            return False

    def test_admin_review_contribution(self, contribution_id: int) -> bool:
        try:
            resp = self._post_json(
                f"/contributions/{contribution_id}/review",
                use_admin=True,
                json={"status": 2, "review_note": "E2E采纳投稿", "points": 2},
            )
            passed = resp.status_code == 200
            self._record("管理员审核投稿", passed, f"status={resp.status_code}, id={contribution_id}")
            return passed
        except Exception as e:
            self._record("管理员审核投稿", False, str(e))
            return False

    def test_admin_list_roles(self) -> bool:
        try:
            resp = self._get("/admin/rbac/roles", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取角色列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取角色列表", False, str(e))
            return False

    def test_admin_list_role_permissions(self) -> bool:
        try:
            resp = self._get("/admin/rbac/roles/permissions", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取角色权限映射", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取角色权限映射", False, str(e))
            return False

    def test_admin_list_permissions(self) -> bool:
        try:
            resp = self._get("/admin/rbac/permissions", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取权限列表", passed, f"status={resp.status_code}")
            return passed
        except Exception as e:
            self._record("管理员获取权限列表", False, str(e))
            return False

    def test_admin_get_user_permissions(self, user_id: int) -> bool:
        try:
            resp = self._get(f"/admin/rbac/users/{user_id}/permissions", use_admin=True)
            passed = resp.status_code == 200
            self._record("管理员获取用户权限", passed, f"status={resp.status_code}, user_id={user_id}")
            return passed
        except Exception as e:
            self._record("管理员获取用户权限", False, str(e))
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

    def test_admin_grant_feature(self, feature_key: str, user_id: int) -> bool:
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

    def test_admin_revoke_feature(self, feature_key: str, user_id: int) -> bool:
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

    def test_admin_get_user_features(self, user_id: int) -> bool:
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
        contribution_id = None
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
        elif self.token:
            self.test_refresh_token()
            self.test_logout_temp_user()
            self.test_logout_all_temp_user()

        # 公开接口
        print("\n🌐 公开接口测试")
        print("-" * 40)
        self.test_get_reviews_by_teacher()
        self.test_get_config_by_key()
        self.test_list_heroes()

        if self.token:
            # 用户接口
            print("\n👤 用户接口测试")
            print("-" * 40)
            self.test_get_profile()
            self.test_update_profile()
            self.test_get_login_days()

            # 通知接口
            print("\n📣 通知接口测试")
            print("-" * 40)
            self.test_get_notifications()
            self.test_get_categories()
            self.test_get_notification_detail()

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
            self.test_get_course_table_bind_count()

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
            contribution_title = self.test_create_contribution()
            self.test_get_contributions()
            self.test_get_user_contribution_stats()
            contribution_id = self.test_get_contribution_detail(contribution_title) if contribution_title else None

            # 倒数日接口
            print("\n⏰ 倒数日接口测试")
            print("-" * 40)
            countdown_id = self.test_create_countdown()
            self.test_get_countdowns()
            if countdown_id:
                self.test_get_countdown_detail(countdown_id)
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
                self.test_get_study_task_detail(task_id)
                self.test_update_study_task(task_id)
                self.test_delete_study_task(task_id)

            print("\n📚 资料接口测试")
            print("-" * 40)
            self.test_get_material_categories()
            material_md5 = self.test_get_materials()
            self.test_get_top_materials()
            self.test_get_material_hot_words()
            self.test_search_materials()
            if material_md5:
                self.test_get_material_detail(material_md5)
                self.test_rate_material(material_md5)
                self.test_download_material(material_md5)

            print("\n🧠 刷题接口测试")
            print("-" * 40)
            project_id = self.test_get_question_projects()
            if project_id:
                question_id = self.test_get_question_list(project_id)
                self.test_get_project_online(project_id)
                if question_id:
                    self.test_get_question_detail(question_id)
                    self.test_record_question_study(question_id, project_id)
                    self.test_submit_question_practice(question_id, project_id)

            print("\n🍅 番茄钟与统计测试")
            print("-" * 40)
            self.test_get_pomodoro_count()
            self.test_increment_pomodoro()
            self.test_get_pomodoro_ranking()
            self.test_get_system_online()
            self.test_get_dictionary_word()

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
            if self.admin_user_id and self.operator_user_id and self.basic_user_id and self.operator_token:
                admin_phone = f"1390000{self.admin_user_id:04d}"
                operator_phone = f"1391000{self.operator_user_id:04d}"
                strong_password = "Admin1234"
                operator_password = "Operator1234"

                print("\n🔑 后台登录接口测试")
                print("-" * 40)
                self.test_admin_set_login_credentials(
                    "管理员设置operator后台凭据",
                    caller_token=self.admin_token,
                    target_user_id=self.operator_user_id,
                    phone=operator_phone,
                    password=operator_password,
                )
                self.test_admin_set_login_credentials(
                    "管理员设置admin后台凭据",
                    caller_token=self.admin_token,
                    target_user_id=self.admin_user_id,
                    phone=admin_phone,
                    password=strong_password,
                )
                self.test_admin_set_login_credentials(
                    "弱密码被拒绝",
                    caller_token=self.admin_token,
                    target_user_id=self.operator_user_id,
                    phone=operator_phone,
                    password="weakpass",
                    expected_status=400,
                )
                self.test_admin_set_login_credentials(
                    "普通用户不能配置后台凭据",
                    caller_token=self.admin_token,
                    target_user_id=self.basic_user_id,
                    phone=f"1392000{self.basic_user_id:04d}",
                    password="Basic1234",
                    expected_status=400,
                )
                self.test_admin_set_login_credentials(
                    "后台手机号冲突被拒绝",
                    caller_token=self.admin_token,
                    target_user_id=self.operator_user_id,
                    phone=admin_phone,
                    password="Conflict1234",
                    expected_status=409,
                )
                self.test_admin_set_login_credentials(
                    "非管理员不能配置后台凭据",
                    caller_token=self.operator_token,
                    target_user_id=self.operator_user_id,
                    phone=operator_phone,
                    password=operator_password,
                    expected_status=403,
                )
                self.test_admin_password_login(
                    "后台登录错误密码",
                    phone=admin_phone,
                    password="WrongPass123",
                    expected_status=401,
                )
                self.admin_password_login_token = self.test_admin_password_login(
                    "admin后台手机号密码登录",
                    phone=admin_phone,
                    password=strong_password,
                )
                self.operator_password_login_token = self.test_admin_password_login(
                    "operator后台手机号密码登录",
                    phone=operator_phone,
                    password=operator_password,
                )
                if self.admin_password_login_token:
                    self.test_admin_user_detail_with_token(
                        "admin后台token访问用户详情",
                        token=self.admin_password_login_token,
                        user_id=self.admin_user_id,
                    )
                if self.operator_password_login_token:
                    self.test_admin_notifications_with_token(
                        "operator后台token访问通知列表",
                        token=self.operator_password_login_token,
                    )

            self.test_admin_get_reviews()
            self.test_admin_get_notifications()
            self.test_admin_get_notification_stats()
            self.test_admin_search_heroes()
            self.test_admin_search_configs()
            self.test_admin_list_roles()
            self.test_admin_list_role_permissions()
            self.test_admin_list_permissions()

            # 功能白名单接口
            print("\n🎯 功能白名单接口测试")
            print("-" * 40)
            feature_key = self.test_admin_create_feature()
            self.test_admin_list_features()
            if feature_key:
                self.test_admin_update_feature(feature_key)
                if self.basic_user_id:
                    self.test_admin_grant_feature(feature_key, user_id=self.basic_user_id)
                    self.test_admin_list_whitelist(feature_key)
                    self.test_admin_get_user_features(user_id=self.basic_user_id)
                    self.test_admin_revoke_feature(feature_key, user_id=self.basic_user_id)
                self.test_admin_delete_feature(feature_key)

            print("\n🛠️ 管理资源接口测试")
            print("-" * 40)
            if self.notification_category_id:
                self.test_admin_update_category(self.notification_category_id)
            hero_id = self.test_admin_create_hero()
            if hero_id:
                self.test_admin_update_hero(hero_id)
                self.test_admin_delete_hero(hero_id)
            config_key = self.test_admin_create_config()
            if config_key:
                self.test_admin_update_config(config_key)
                self.test_get_config_by_specific_key(config_key)
                self.test_admin_delete_config(config_key)
            notification_id = self.test_operator_create_notification()
            if notification_id:
                self.test_admin_get_notification_detail(notification_id)
                self.test_admin_update_notification(notification_id)
                self.test_admin_approve_notification(notification_id)
                self.test_admin_publish_notification(notification_id)
                self.test_admin_pin_notification(notification_id)
                self.test_admin_unpin_notification(notification_id)
                self.test_admin_delete_notification(notification_id)
            if self.basic_user_id:
                self.test_admin_grant_points(self.basic_user_id)
                self.test_admin_get_user_points(self.basic_user_id)
                self.test_admin_get_user_point_stats(self.basic_user_id)
                self.test_admin_get_user_point_transactions(self.basic_user_id)
                self.test_admin_get_user_permissions(self.basic_user_id)
                if self.token:
                    self.test_spend_points()
            self.test_admin_get_contribution_stats()
            if contribution_id:
                self.test_admin_review_contribution(contribution_id)
            self.test_admin_get_countdown_group_stats()
            self.test_admin_get_studytask_group_stats()
            self.test_admin_get_gpa_backup_group_stats()

            print("\n🧩 新增后台接口测试")
            print("-" * 40)
            self.test_admin_list_coursetables()
            if self.basic_user_id:
                self.test_admin_reset_coursetable_bind_count(self.basic_user_id)
            coursetable_id = self.test_admin_create_coursetable()
            if coursetable_id:
                self.test_admin_get_coursetable_detail(coursetable_id)
                self.test_admin_update_coursetable(coursetable_id)
                self.test_admin_delete_coursetable(coursetable_id)

            self.test_admin_list_failrates()
            failrate_id = self.test_admin_create_failrate()
            if failrate_id:
                self.test_admin_get_failrate_detail(failrate_id)
                self.test_admin_update_failrate(failrate_id)
                self.test_admin_delete_failrate(failrate_id)

            self.test_admin_list_question_projects()
            project_id = self.test_admin_create_question_project()
            if project_id:
                self.test_admin_get_question_project_detail(project_id)
                self.test_admin_update_question_project(project_id)
                self.test_admin_list_questions(project_id=project_id)
                question_id = self.test_admin_create_question(project_id)
                if question_id:
                    self.test_admin_get_question_detail(question_id)
                    self.test_admin_update_question(question_id)
                    self.test_admin_delete_question(question_id)
                self.test_admin_delete_question_project(project_id)

            if self.token:
                print("\n📚 绩点备份接口测试")
                print("-" * 40)
                gpa_backup_id = self.test_create_gpa_backup()
                self.test_list_gpa_backups()
                if gpa_backup_id:
                    self.test_get_gpa_backup_detail(gpa_backup_id)
                    self.test_delete_gpa_backup(gpa_backup_id)

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
