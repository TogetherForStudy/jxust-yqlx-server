# User Module Design

用户模块的设计是构建用户管理和认证系统的重要部分。

由于我们的用户信息来自外部系统(WeChat),我们不需要存储用户的密码等敏感信息。

因此,用户模块主要负责用户的登录、信息更新和权限管理。

## 文档更新记录

| Code | Module | Date       | Author | PRI | Description |
|------|--------|------------|--------|-----|-------------|
| 1    | init   | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建    |
| 2    | update | 2025-06-22 | AEnjoy | P0  | 添加用户信息权限接口 |

## 设计原则

1. **高内聚低耦合**: 用户模块与其他模块通过接口和事件解耦，确保模块独立性(可以独立测试和部署)
2. **安全性**: 确保用户信息的安全性，防止未授权访问
3. **易用性**: 提供简单易用的用户接口和权限管理功能

## 模型

表元信息已省略

### 用户:

用户表:(OpenID, UnionID, RoleControlTag, IsActive)

用户信息表:(UnionID, Nickname, AvatarURL, Email, Phone)

学生详细信息表:(UnionID, PeopleID(not null), Name, College, Meta(JSON))

    Meta:(Class, Major, Grade)

教师详细信息表:(UnionID, PeopleID(null), Name, College, Meta(JSON))

    Meta:(Department, Title, Office, Status, Description)

学生信息表和教师信息表可以合并

## API

1. P0-**用户信息接口**:

   - 获取信息接口(从上下文中获取union_id): GET `/api/<version>/user/info`
   - 获取用户信息接口(一般是获取教师信息): GET `/api/<version>/user/info/<union_id>`
   - 更新信息接口: POST `/api/<version>/user/info[/<union_id>]`
   
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
   { }
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
   
   获取用户信息流程：
   
   ```pseudocode
   func getUserInfo(req,resp){
      unionID := req.GetUnionIDFromUrl()
      if unionID == "" {
          unionID = req.GetUnionIDFromContext()
      }
      userRows := database.Query("SELECT * FROM user_table WHERE union_id = ?", unionID)
      if len(userRows) == 0 {
            resp.SetStatusCode(404)
            resp.SetStatusMessage("User(teacher) not found")
            return
      }
      userInfo := userRows[0]
      
      resp.SetStatusCode(200)
      resp={
       "union_id": userInfo.UnionID,
       "nickname": userInfo.Nickname,
       "avatar_url": userInfo.AvatarURL,
       "email": userInfo.Email,
       "phone": userInfo.Phone,
       "name": userInfo.Name,
       "college": userInfo.College,
       "meta": userInfo.Meta
      }
   }
   ```
   
   更新用户信息流程：
   
   ```pseudocode
   func updateUserInfo(req, resp) {
      unionID := req.GetUnionIDFromUrl()
      if unionID == "" {
          unionID = req.GetUnionIDFromContext()
      }else if ctxID:=req.GetUnionIDFromContext();unionID != ctxID{ // 如果提供了union_id，且该union_id与上下文不一致,则必须要做鉴权
          ctrl:= user.GetRoleControlTags(ctxID)['UpdateUserInfo']
          userPermission := user.GetUnionPermissionId(unionID)
          if !ctrl.HasPermission(userPermission) {
                resp.SetStatusCode(403)
                resp.SetStatusMessage("Forbidden: You do not have permission to update this user")
                return
          }
      }
  
       userRows := database.Query("SELECT * FROM user_table WHERE union_id = ?", unionID)
       if len(userRows) == 0 {
           resp.SetStatusCode(404)
           resp.SetStatusMessage("User(teacher) not found")
           return
       }
       
       userInfo := userRows[0]
       
       // 更新用户信息
       if req.Nickname != nil {
           userInfo.Nickname = req.Nickname
       }
       if req.AvatarURL != nil {
           userInfo.AvatarURL = req.AvatarURL
       }
       if req.Email != nil {
           userInfo.Email = req.Email
       }
       if req.Phone != nil {
           userInfo.Phone = req.Phone
       }
       if req.Name != nil {
           userInfo.Name = req.Name
       }
       if req.College != nil {
           userInfo.College = req.College
       }
       
       // 更新数据库
       database.Execute("UPDATE user_table SET nickname=?, avatar_url=?, email=?, phone=?, name=?, college=? WHERE union_id=?",
                        userInfo.Nickname, userInfo.AvatarURL, userInfo.Email, userInfo.Phone, userInfo.Name, userInfo.College, unionID)
       
       resp.SetStatusCode(200)
       resp = {}
   }
   ```

2. P0-**用户登录接口**:

   - 登录接口: POST `/api/<version>/user/auth/wechat-login`
   
   LoginRequest:
   ```json
   {
       "code": "string"
   }
   ```
   
   LoginResponse:
   ```json
   {
       "union_id": "string",
       "token": "string"
   }
   ```
   
   登录流程：
   
   ```pseudocode
   func login(req, resp) {
       code := req.GetCode()
       if code == "" {
           resp.SetStatusCode(400)
           resp.SetStatusMessage("Bad Request: Code is required")
           return
       }
       
       // 调用微信API获取用户信息
       userInfo := wechat.GetUserInfoByCode(code)
       if userInfo == nil {
           resp.SetStatusCode(401)
           resp.SetStatusMessage("Unauthorized: Invalid code")
           return
       }
       
       // 检查用户是否存在
       userRows := database.Query("SELECT * FROM user_table WHERE union_id = ?", userInfo.UnionID)
       if len(userRows) == 0 {
           // 如果不存在，则创建新用户
           database.Execute("INSERT INTO user_table (union_id, nickname, avatar_url, email, phone) VALUES (?, ?, ?, ?, ?)",
                            userInfo.UnionID, userInfo.Nickname, userInfo.AvatarURL, userInfo.Email, userInfo.Phone)
       }
       
       // 生成JWT token
       token := auth.GenerateToken(userInfo.UnionID)
       
       resp.SetStatusCode(200)
       resp.Headers["Authorization"] = "Bearer " + token
       resp = {
           "union_id": userInfo.UnionID,
           "token": token
       }
   }
   ```

3. P1-**设置用户权限组接口(对内部使用)**:

    - 设置权限组接口: POST `/api/<version>/user/role`
    
    SetRoleRequest:
    ```json
    {
         "union_id": "string",
         "role_control_tag": "string"
    }
    ```
    
    SetRoleResponse(Success):
    ```json
    { }
    ```
    
    设置用户权限组流程：
    
    ```pseudocode
    func setUserRole(req, resp) {
         ctxUnionID := req.GetUnionIDFromContext()
         // 检查当前用户是否有权限设置其他用户的角色      
         ctrl := user.GetRoleControlTags(ctxUnionID)['SetUserRole']
         userPermission := user.GetUnionPermissionId(reqUnionID)
         if !ctrl.HasPermission(userPermission) {
                resp.SetStatusCode(403)
                resp.SetStatusMessage("Forbidden: You do not have permission to set this user's role")
                return
         }
   
         reqUnionID := req.GetUnionID()
         reqRoleControlTag := req.GetRoleControlTag()
         
         if reqUnionID == "" || reqRoleControlTag == "" {
              resp.SetStatusCode(400)
              resp.SetStatusMessage("Bad Request: UnionID and RoleControlTag are required")
              return
         }   
         // 更新用户角色控制标签
         database.Execute("UPDATE user_table SET role_control_tag = ? WHERE union_id = ?", roleControlTag, unionID)
         
         resp.SetStatusCode(200)
         resp = {}
    }
    ```
4. P1-**获取用户权限接口(对内部使用)**:
   
   - 获取用户权限接口: GET `/api/<version>/user/permissions[?union_id=<union_id>]`
   
   GetPermissionsRequest(nobody)-GetPermissionsResponse:
   
   GetPermissionsResponse:
   ```json
   {
       "union_id": "string",
       "permissions": ["string"]
   }
   ```
   
   获取用户权限流程：
   
    ```pseudocode
    func getUserPermissions(req, resp) {
        unionID := req.GetUnionIDFromQuery()
        if unionID == "" {
            unionID = req.GetUnionIDFromContext()
        }
        if ctxID:= req.GetUnionIDFromContext() ; unionID != ctxID { // 如果提供了union_id，且该union_id与上下文不一致,则必须要做鉴权
            ctrl := user.GetRoleControlTags(ctxID)['GetUserPermissions']
            userPermission := user.GetUnionPermissionId(unionID)
            if !ctrl.HasPermission(userPermission) {
                resp.SetStatusCode(403)
                resp.SetStatusMessage("Forbidden: You do not have permission to get this user's permissions")
                return
            }
        }
   
        // 查询用户角色和权限
        roles := database.Query("SELECT role_control_tag FROM user_table WHERE union_id = ?", unionID)
        if len(roles) == 0 {
            resp.SetStatusCode(404)
            resp.SetStatusMessage("User not found")
            return
        }
        
        permissions := user.GetPermissionsByRoles(roles)
        
        resp.SetStatusCode(200)
        resp = {
            "union_id": unionID,
            "permissions": permissions
        }
    }
    ```