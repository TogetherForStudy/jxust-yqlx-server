# API Docs (Old Version)

新版本Api文档请参照设计文档和apifox接口文档.

## API 文档

### 认证相关

#### 微信小程序登录
```http
POST /api/v0/auth/wechat-login
Content-Type: application/json

{
  "code": "微信授权码"
}
```

#### 获取用户资料
```http
GET /api/v0/user/profile
Authorization: Bearer <JWT_TOKEN>
```

#### 更新用户资料
```http
PUT /api/v0/user/profile
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
GET /api/v0/teachers/search?keyword=教师姓名
```

### 评价相关

#### 创建教师评价
```http
POST /api/v0/reviews
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
GET /api/v0/reviews/teacher?teacher_name=教师姓名&page=1&size=10
```

#### 获取用户评价记录
```http
GET /api/v0/reviews/user?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```

### 存储相关

#### 获取文件URL
```http
GET /api/v0/store/{resource_id}/url
Authorization: Bearer <JWT_TOKEN>
```

#### 获取文件流
```http
GET /api/v0/store/{resource_id}/stream
Authorization: Bearer <JWT_TOKEN>
```

### 管理员功能

#### 获取评价列表
```http
GET /api/v0/admin/reviews?page=1&size=10&status=1
Authorization: Bearer <JWT_TOKEN>
```

#### 审核通过评价
```http
POST /api/v0/admin/reviews/{id}/approve
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "admin_note": "审核备注"
}
```

#### 审核拒绝评价
```http
POST /api/v0/admin/reviews/{id}/reject
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "admin_note": "拒绝理由"
}
```

#### 上传文件
```http
POST /api/v0/store
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data

# Body is form-data
# file: (binary)
# tags: {"key": "value"}
```

#### 删除文件
```http
DELETE /api/v0/store/{resource_id}
Authorization: Bearer <JWT_TOKEN>
```

#### 获取文件列表
```http
GET /api/v0/store?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```

#### 获取过期文件列表
```http
GET /api/v0/store/expired?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```
