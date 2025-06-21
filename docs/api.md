# API Docs (Old Version)

新版本Api文档请参照设计文档和apifox接口文档.

## API 文档

### 认证相关

#### 微信小程序登录
```http
POST /api/auth/wechat-login
Content-Type: application/json

{
  "code": "微信授权码"
}
```

#### 获取用户资料
```http
GET /api/user/profile
Authorization: Bearer <JWT_TOKEN>
```

#### 更新用户资料
```http
PUT /api/user/profile
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "nickname": "昵称",
  "avatar": "头像URL",
  "phone": "手机号",
  "student_id": "学号",
  "real_name": "真实姓名",
  "college": "学院",
  "major": "专业",
  "grade": "年级"
}
```

### 教师相关

#### 搜索教师
```http
GET /api/teachers/search?keyword=教师姓名
```

### 评价相关

#### 创建教师评价
```http
POST /api/reviews
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "teacher_name": "教师姓名",
  "course": "课程名称",
  "content": "评价内容（200字以内）",
  "rating": 5,
  "is_anonymous": false
}
```

#### 查询教师评价
```http
GET /api/reviews/teacher?teacher_name=教师姓名&page=1&size=10
```

#### 获取用户评价记录
```http
GET /api/reviews/user?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```

### 管理员功能

#### 获取评价列表
```http
GET /api/admin/reviews?page=1&size=10&status=1
Authorization: Bearer <JWT_TOKEN>
```

#### 审核通过评价
```http
POST /api/admin/reviews/{id}/approve
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "admin_note": "审核备注"
}
```

#### 审核拒绝评价
```http
POST /api/admin/reviews/{id}/reject
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "admin_note": "拒绝理由"
}
```
