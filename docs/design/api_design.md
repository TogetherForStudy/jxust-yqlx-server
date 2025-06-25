# API Request/Response Design:

## Restful API

### 设计原则

对于所有的Restful API接口，遵循以下设计原则：

1. **清晰的资源路径**: 使用名词表示资源，避免动词，符合RESTful风格。
2. **HTTP方法语义明确**: 使用GET、POST、PUT、DELETE等HTTP方法表示操作类型。
3. **版本化接口**: 所有API接口都应包含版本号，便于未来扩展和兼容。
4. **统一的请求和响应格式**: 所有API请求(除GET方法)和响应(除二进制内容输出)都应使用JSON格式，包含必要的状态码和错误信息。
5. **错误处理**: 提供统一的错误响应格式，包含错误码和错误信息，便于前端处理。

### URI

所有API接口的URI应遵循以下格式：

```
/api/<version>/<module-name>/<resources...>
```

- version: API版本号，例如`v0` `v1` `v2`等,现阶段采用`v0`,代表接口处于研发中。
- module-name: 模块名称，例如`user` `order` `product`等。
- resources...: 资源路径，可以是单个资源或资源集合，例如`/user/info`表示用户信息，`/order/list`表示订单列表。

协议总体要求

- 报文统一采用UTF-8编码，**无压缩方式**传输。
- HTTP请求方式：GET/POST/PUT/DELETE
- 请求（非GET）和应答消息（非二进制）的报文体统一采用JSON文本格式。
- JSON报文中的数值型字段，统一按照十进制方式进行序列化处理，浮点内容采用64位浮点

### 请求报文头:

| Header 标识符   | 类型     | 是否必填        | 描述                                               |
|--------------|--------|-------------|--------------------------------------------------|
| Content-Type | String | 是(对于非GET方法) | 请求报文体的内容类型，一般要求为：application/json; charset=UTF-8 |
| Content-Type | String | 否(对于GET方法)  | charset=UTF-8                                    |
| X-Request-ID | String | 否           | 请求的唯一标识符，用于追踪请求和响应，建议使用UUID格式,如果不填,将有中间件生成       |

### 应答报文头:

| Header 标识符   | 类型     | 是否必填 | 描述                                               |
|--------------|--------|------|--------------------------------------------------|
| Content-Type | String | 是    | 应答报文体的内容类型，一般要求为：application/json; charset=UTF-8 |

### 应答报文体:

  应答报文体可以为空，客户端此时通过HTTP Response Status来判断接口调用是否正常；如果接口中需要返回数据应答，则应答报文体的内容为标准JSON格式，各个接口的应答报文体的详细定义请参见各个接口中的应答报文体的定义。
  
### 标准请求体要求：

  无，根据各接口的定义要求
  
### 标准响应要求：
  
需要有一个规范的外部结构(提供StatusCode，StatusMessage，RequestId)，然后再返回实际内容（Resutl）：
```json
{
  "StatusCode": 0,
  "StatusMessage": "Success",
  "RequestId": "xxxxxxx",
  "Result": {}
}
```

解释：
  RequestId 为传入的`X-Request-ID`，如果没有传入，则在接口层必须随机生成一个，用于在后续过程中根据日志调试程序

- 若接口一切正常，StatusCode = 0，StatusMessage = Success
- 若业务失败，HTTPCode返回200，StatusCode返回业务错误码，StatusMessage返回错误信息、
- 若找不到，路由需返回HTTPCode404，并返回JSON响应
- 若鉴权失败，路由需返回HTTPCode401，并返回JSON响应
- 若后端异常，HTTPCode返回5xx，此时不要求接口返回json内容

业务码：参照 [模块设计](module_common_design.md)
