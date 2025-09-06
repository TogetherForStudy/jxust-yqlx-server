## GoJxust API 文档（v0）

说明：所有接口返回统一响应壳：
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "<trace id>",
  "Result": <具体数据>
}
```

鉴权说明：
- 公开接口：无 Token
- 需认证：`Authorization: Bearer <JWT_TOKEN>`
- 需管理员：在“需认证”的基础上，用户 `role=admin`

基础路径：`/api/v0`

### 健康检查
```http
GET /health
```

### 认证
- 微信登录
```http
POST /api/v0/auth/wechat-login
Content-Type: application/json

Body(dto): request.WechatLoginRequest
{
  "code": "string"
}

Result(dto): response.WechatLoginResponse
{
  "token": "string",
  "user_info": models.User
}
```

- 模拟微信登录（测试）
```http
POST /api/v0/auth/mock-wechat-login
Content-Type: application/json

Body(dto): request.MockWechatLoginRequest
{
  "test_user": "normal | admin | new_user"
}

Result(dto): response.WechatLoginResponse
```

### 用户
- 获取当前用户资料（需认证）
```http
GET /api/v0/user/profile
Authorization: Bearer <JWT_TOKEN>

Result: models.User
```

- 更新当前用户资料（需认证）
```http
PUT /api/v0/user/profile
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.UpdateProfileRequest
{
  "nickname": "string",
  "avatar": "string",
  "phone": "string",
  "student_id": "string",
  "real_name": "string",
  "college": "string",
  "major": "string",
  "class_id": "string"
}
```

### 评价
- 公开获取教师评价
```http
GET /api/v0/reviews/teacher?teacher_name=xx&page=1&size=10

Result: utils.PageResponse{data=[]models.TeacherReview}
```

- 创建教师评价（需认证）
```http
POST /api/v0/reviews/
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.CreateReviewRequest
{
  "teacher_name": "string",
  "campus": "string",
  "course_name": "string",
  "content": "string(<=200)",
  "attitude": 1 | 2 | 3
}
```

- 获取我的评价记录（需认证）
```http
GET /api/v0/reviews/user?page=1&size=10
Authorization: Bearer <JWT_TOKEN>

Result: utils.PageResponse{data=[]models.TeacherReview}
```

- 管理员获取评价列表（需管理员）
```http
GET /api/v0/reviews/?page=1&size=10&teacher_name=xx&status=1
Authorization: Bearer <JWT_TOKEN>

Result: utils.PageResponse{data=[]models.TeacherReview}
```

- 审核通过（需管理员）
```http
POST /api/v0/reviews/{id}/approve
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): { "admin_note": "string" }
```

- 审核拒绝（需管理员）
```http
POST /api/v0/reviews/{id}/reject
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): { "admin_note": "string(必填)" }
```

- 删除评价（需管理员）
```http
DELETE /api/v0/reviews/{id}
Authorization: Bearer <JWT_TOKEN>
```

### 课程表（需认证）
- 获取用户课程表
```http
GET /api/v0/coursetable?semester=2024-2025-1

Result(dto): response.CourseTableResponse
{
  "class_id": "string",
  "semester": "string",
  "course_data": "json"
}
```

- 搜索班级
```http
GET /api/v0/coursetable/search?keyword=xx&page=1&size=10

Result(dto): response.SearchClassResponse
{
  "list": [{"class_id":"string","semester":"string"}],
  "total": 0,
  "page": 1,
  "size": 10
}
```

- 更新用户班级
```http
PUT /api/v0/coursetable/class
Content-Type: application/json

Body(dto): request.UpdateUserClassRequest
{ "class_id": "string" }
```

- 编辑个人课表单个格子
```http
PUT /api/v0/coursetable
Content-Type: application/json

Body(dto): request.EditCourseCellRequest
{
  "semester": "string",
  "index": "1-35",
  "value": { ... 任意JSON ... }
}
```

### 挂科率（需认证）
- 搜索
```http
GET /api/v0/failrate/search?keyword=xx&page=1&size=10

Result(dto): response.FailRateListResponse
{
  "list": [{"id":1,"course_name":"","department":"","semester":"","failrate":0}],
  "total": 0,
  "page": 1,
  "size": 10
}
```

- 随机Top10
```http
GET /api/v0/failrate/rand

Result(dto): response.FailRateListResponse
{
  "list": [ ... 同上 ... ]
}
```

### Heroes（需认证）
- 列表（全部，按 sort 升序）
```http
GET /api/v0/heroes/

Result: []string
```

- 创建（需管理员）
```http
POST /api/v0/heroes/
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.CreateHeroRequest
{ "name": "string", "sort": 0, "is_show": true }
```

- 更新（需管理员）
```http
PUT /api/v0/heroes/{id}
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.UpdateHeroRequest
{ "name": "string", "sort": 0, "is_show": true }
```

- 删除（需管理员）
```http
DELETE /api/v0/heroes/{id}
Authorization: Bearer <JWT_TOKEN>
```

### 系统配置
- 公开读取
```http
GET /api/v0/config/{key}

Result: { "key": "string", "value": "string", "value_type": "string", "description": "string" }
```

- 创建（需管理员）
```http
POST /api/v0/config/
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.CreateConfigRequest
{ "key":"string","value":"string","value_type":"string|number|boolean|json","description":"string" }
```

- 更新（需管理员）
```http
PUT /api/v0/config/{key}
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

Body(dto): request.UpdateConfigRequest
{ "value":"string","value_type":"string|number|boolean|json","description":"string" }
```

- 删除（需管理员）
```http
DELETE /api/v0/config/{key}
Authorization: Bearer <JWT_TOKEN>
```

### 存储相关(需认证)

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

#### 上传文件（管理员）
```http
POST /api/v0/store
Authorization: Bearer <JWT_TOKEN>
Content-Type: multipart/form-data

# Body is form-data
# file: (binary)
# tags: {"key": "value"}
```

**Example using curl:**
```bash
curl -X POST \
  http://localhost:8085/api/v0/store \
  -H "Authorization: Bearer <YOUR_JWT_TOKEN>" \
  -F "file=@/path/to/your/file.jpg" \
  -F 'tags={"source":"test-script"}'
```

#### 删除文件（管理员）
```http
DELETE /api/v0/store/{resource_id}
Authorization: Bearer <JWT_TOKEN>
```

#### 获取文件列表（管理员）
```http
GET /api/v0/store/list?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```

#### 获取过期文件列表（管理员）
```http
GET /api/v0/store/expired?page=1&size=10
Authorization: Bearer <JWT_TOKEN>
```
