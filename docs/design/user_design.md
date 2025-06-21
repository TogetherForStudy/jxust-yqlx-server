# User Module Design

用户模块的设计是构建用户管理和认证系统的重要部分。

由于我们的用户信息来自外部系统(WeChat),我们不需要存储用户的密码等敏感信息。

因此,用户模块主要负责用户的登录、信息更新和权限管理。

## 文档更新记录

| Code | Module | Date       | Author | PRI | Description |
|------|--------|------------|--------|-----|-------------|
| 1    | init   | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建    |

## 设计原则

1. **高内聚低耦合**: 用户模块与其他模块通过接口和事件解耦，确保模块独立性(可以独立测试和部署)
2. **安全性**: 确保用户信息的安全性，防止未授权访问
3. **易用性**: 提供简单易用的用户接口和权限管理功能

## 模型

表元信息已省略

### 用户:

用户表:(OpenID, UnionID, RoleControlTags(JSON))

    RoleControlTags: a json: key is RoleControlTag, value is an `UnionPermissionId`

用户信息表:(UnionID, Nickname, AvatarURL, Email, Phone)

学生信息表:(UnionID, PeopleID(not null), Name, College, Meta(JSON))

    Meta:(Class, Major, Grade)

教师信息表:(UnionID, PeopleID(null), Name, College, Meta(JSON))

    Meta:(Department, Title, Office, Status, Description)

学生信息表和教师信息表可以合并

### 角色控制模型:

角色控制表:(RoleControlTag, Description, Permissions(JSON))

    Permissions:(PermissionTag, Description)

授予关系:(UnionPermissionId, Bool)

### 权限模型:

权限表:(PermissionTag, Description)

## API

1. P0-**用户信息接口**:

   - 获取信息接口: GET `/api/<version>/user/info`
   - 更新信息接口: POST `/api/<version>/user/info`

GetUserInfoRequest(nobody)-GetUserInfoResponse:

UpdateUserInfoRequest-UpdateUserInfoResponse:

对于GetUserInfoResponse,所有字段不可空;对于UpdateUserInfoRequest,所有字段除`union_id`外均可空

UserInfo:(GetUserInfoResponse and UpdateUserInfoRequest)

```json
{
    "union_id": "string",
    "nickname": "string",
    "avatar_url": "string",
    "email": "string",
    "phone": "string",
    "name": "string",
    "college": "string",
    "meta": {}
}
```

UpdateUserInfoResponse:
```json
{
	
}
```

```bash
# 获取用户信息
curl -X GET http://example.com/api/v0/user/info \
     -H "X-Request-ID: uuid" \
     -H "Authorization: Bearer token"
# 输出
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "uuid",
  "Result": {
    "union_id": "string",
    "nickname": "string",
    "avatar_url": "string",
    "email": "string",
    "phone": "string",
    "name": "string",
    "college": "string",
    "meta": {}
}
}
```

更新用户信息

```bash
curl -X POST http://example.com/api/v0/user/info \
     -H "Content-Type: application/json" \
     -H "X-Request-ID: uuid" \
     -H "Authorization : Bearer token"\
     -d '{
    "union_id": "string",
    "nickname": "string",
    "avatar_url": "string",
    "email": "string",
    "phone": "string",
    "name": "string",
    "college": "string",
    "meta": {}
}'
# 输出
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "uuid",
  "Result": {}
}
```

todos:...

