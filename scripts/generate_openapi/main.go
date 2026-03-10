package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	chatdto "github.com/TogetherForStudy/jxust-yqlx-server/internal/dto"
	req "github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	resp "github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
)

type schemaBuilder func(*generator) map[string]any

type responseMode int

const (
	responseEnvelope responseMode = iota
	responseRawJSON
	responseBinary
	responseSSE
)

type parameterSpec struct {
	Name        string
	In          string
	Required    bool
	Description string
	Schema      schemaBuilder
}

type operation struct {
	Method             string
	Path               string
	Tag                string
	Summary            string
	Description        string
	Security           bool
	Permission         string
	DevOnly            bool
	Idempotent         bool
	RouterMethod       string
	QueryType          reflect.Type
	Parameters         []parameterSpec
	RequestBody        schemaBuilder
	RequestContentType string
	ResponseMode       responseMode
	Success            schemaBuilder
	SuccessContentType string
	ErrorStatus        []int
}

type operationOption func(*operation)

type objectField struct {
	name   string
	schema schemaBuilder
}

type generator struct {
	schemas map[string]map[string]any
}

var timeType = reflect.TypeOf(time.Time{})

func main() {
	g := newGenerator()
	doc := g.buildDocument()

	outputPath := filepath.Join("docs", "openapi", "openapi.json")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		log.Fatalf("marshal openapi: %v", err)
	}
	content = append(content, '\n')

	if err := os.WriteFile(outputPath, content, 0o644); err != nil {
		log.Fatalf("write openapi file: %v", err)
	}

	log.Printf("generated %s", outputPath)
}

func newGenerator() *generator {
	g := &generator{
		schemas: map[string]map[string]any{},
	}
	g.addCustomSchemas()
	return g
}

func (g *generator) addCustomSchemas() {
	g.schemas["ErrorResponse"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"StatusCode": map[string]any{
				"type":        "integer",
				"format":      "int32",
				"description": "业务状态码，非 0 表示异常",
				"example":     11002,
			},
			"StatusMessage": map[string]any{
				"type":        "string",
				"description": "业务错误信息",
				"example":     "无效的 Authorization 头",
			},
			"RequestId": map[string]any{
				"type":        "string",
				"description": "请求唯一标识",
				"example":     "req-01hxyzexample",
			},
		},
		"required": []string{"StatusCode", "StatusMessage", "RequestId"},
	}

	g.schemas["ChatToolCall"] = map[string]any{
		"type":                 "object",
		"description":          "Eino ToolCall 原始结构，字段由上游库控制",
		"additionalProperties": true,
	}
	g.schemas["ChatMessage"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"role": map[string]any{
				"type":        "string",
				"description": "消息角色",
				"example":     "user",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "消息内容",
			},
			"tool_calls": map[string]any{
				"type":  "array",
				"items": refSchema("ChatToolCall"),
			},
			"tool_call_id": map[string]any{
				"type":        "string",
				"description": "工具调用 ID",
			},
		},
	}
}

func (g *generator) buildDocument() map[string]any {
	paths := map[string]any{}
	for _, op := range buildOperations() {
		pathItem, ok := paths[op.Path].(map[string]any)
		if !ok {
			pathItem = map[string]any{}
			paths[op.Path] = pathItem
		}
		pathItem[strings.ToLower(op.Method)] = g.buildOperation(op)
	}

	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "GoJxust API",
			"version":     "1.0.1",
			"description": "根据 `internal/router/router.go`、handler、DTO 与 service 生成的 OpenAPI 文档。默认 JSON 响应使用 `StatusCode/StatusMessage/RequestId/Result` 信封；`/health`、聊天 SSE、文件流、MCP 与 MinIO 代理属于例外。",
		},
		"servers": []any{
			map[string]any{
				"url":         "http://localhost:8080",
				"description": "本地开发环境",
			},
		},
		"tags":  buildTags(),
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"BearerAuth": map[string]any{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
					"description":  "使用 `Authorization: Bearer <access_token>` 传递访问令牌",
				},
			},
			"parameters": map[string]any{
				"XRequestID": map[string]any{
					"name":        "X-Request-ID",
					"in":          "header",
					"required":    false,
					"description": "可选请求 ID。未传入时由服务端自动生成，并同步写入响应头与响应体的 `RequestId`。",
					"schema": map[string]any{
						"type": "string",
					},
				},
				"XIdempotencyKey": map[string]any{
					"name":        constant.IdempotencyKey,
					"in":          "header",
					"required":    false,
					"description": "推荐用于写操作的幂等键；重复提交时可能直接回放缓存结果，并返回 `X-Idempotency-Replayed: true`。",
					"schema": map[string]any{
						"type": "string",
					},
				},
			},
			"schemas": g.schemas,
		},
	}
}

func (g *generator) buildOperation(op operation) map[string]any {
	params := []any{
		map[string]any{"$ref": "#/components/parameters/XRequestID"},
	}
	if op.Idempotent {
		params = append(params, map[string]any{"$ref": "#/components/parameters/XIdempotencyKey"})
	}
	for _, param := range g.queryParameters(op.QueryType) {
		params = append(params, param)
	}
	for _, param := range op.Parameters {
		params = append(params, param.build(g))
	}

	item := map[string]any{
		"tags":        []string{op.Tag},
		"summary":     op.Summary,
		"operationId": buildOperationID(op.Method, op.Path),
		"responses":   g.buildResponses(op),
	}
	if op.Description != "" {
		item["description"] = op.Description
	}
	if len(params) > 0 {
		item["parameters"] = params
	}
	if op.Security {
		item["security"] = []any{map[string]any{"BearerAuth": []any{}}}
	}
	if op.Permission != "" {
		item["x-permission"] = op.Permission
	}
	if op.DevOnly {
		item["x-environment"] = "non-release only"
	}
	if op.RouterMethod != "" {
		item["x-router-method"] = op.RouterMethod
	}
	if op.RequestBody != nil {
		contentType := op.RequestContentType
		if contentType == "" {
			contentType = "application/json"
		}
		item["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				contentType: map[string]any{
					"schema": op.RequestBody(g),
				},
			},
		}
	}
	return item
}

func (g *generator) buildResponses(op operation) map[string]any {
	responses := map[string]any{
		"200": g.successResponse(op),
	}

	statuses := map[int]struct{}{
		400: {},
		500: {},
	}
	if op.Security {
		statuses[401] = struct{}{}
	}
	if op.Permission != "" {
		statuses[403] = struct{}{}
	}
	if op.Idempotent {
		statuses[409] = struct{}{}
	}
	if hasLookupParam(op.Path) {
		statuses[404] = struct{}{}
	}
	for _, status := range op.ErrorStatus {
		statuses[status] = struct{}{}
	}

	var sorted []int
	for status := range statuses {
		sorted = append(sorted, status)
	}
	sort.Ints(sorted)
	for _, status := range sorted {
		responses[strconv.Itoa(status)] = jsonResponse("错误响应", refSchema("ErrorResponse"))
	}

	return responses
}

func (g *generator) successResponse(op operation) map[string]any {
	switch op.ResponseMode {
	case responseRawJSON:
		contentType := op.SuccessContentType
		if contentType == "" {
			contentType = "application/json"
		}
		return map[string]any{
			"description": "成功响应",
			"content": map[string]any{
				contentType: map[string]any{
					"schema": op.Success(g),
				},
			},
		}
	case responseBinary:
		contentType := op.SuccessContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		return map[string]any{
			"description": "二进制流响应",
			"content": map[string]any{
				contentType: map[string]any{
					"schema": map[string]any{
						"type":   "string",
						"format": "binary",
					},
				},
			},
		}
	case responseSSE:
		return map[string]any{
			"description": "Server-Sent Events 流式响应",
			"content": map[string]any{
				"text/event-stream": map[string]any{
					"schema": map[string]any{
						"type":        "string",
						"description": "事件流文本，事件类型包括 start/resume_start/content/reasoning/tool_call/tool_result/interrupt/end。",
					},
				},
			},
		}
	default:
		return jsonResponse("成功响应", envelopeSchema(g, op.Success))
	}
}

func envelopeSchema(g *generator, result schemaBuilder) map[string]any {
	properties := map[string]any{
		"StatusCode": map[string]any{
			"type":        "integer",
			"format":      "int32",
			"description": "业务状态码，成功固定为 0",
			"example":     0,
		},
		"StatusMessage": map[string]any{
			"type":        "string",
			"description": "业务状态说明",
			"example":     "Success",
		},
		"RequestId": map[string]any{
			"type":        "string",
			"description": "请求唯一标识",
		},
	}
	required := []string{"StatusCode", "StatusMessage", "RequestId"}
	if result != nil {
		properties["Result"] = result(g)
		required = append(required, "Result")
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

func jsonResponse(description string, schema map[string]any) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

func buildTags() []any {
	return []any{
		tag("Health", "健康检查"),
		tag("MCP", "LLM Tool Calling / MCP 接口"),
		tag("Auth", "认证与会话"),
		tag("User", "用户信息"),
		tag("Reviews", "教师评价"),
		tag("CourseTable", "课程表"),
		tag("FailRate", "挂科率"),
		tag("Heroes", "英雄榜"),
		tag("Config", "系统配置"),
		tag("Storage", "对象存储与 CDN"),
		tag("Points", "积分"),
		tag("Contributions", "投稿"),
		tag("Countdowns", "倒数日"),
		tag("StudyTasks", "学习任务"),
		tag("Materials", "资料中心"),
		tag("Notifications", "通知与分类"),
		tag("Questions", "刷题"),
		tag("Pomodoro", "番茄钟"),
		tag("Chat", "学习对话"),
		tag("Stats", "统计"),
		tag("Dictionary", "词典"),
		tag("Features", "功能管理"),
		tag("AdminUsers", "管理员用户操作"),
		tag("RBAC", "角色权限管理"),
		tag("Proxy", "MinIO 反向代理"),
	}
}

func (g *generator) queryParameters(t reflect.Type) []any {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var params []any
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		name, defaultValue := formFieldName(field)
		if name == "" || name == "-" {
			continue
		}

		schema := g.schemaForType(field.Type)
		applyBindingConstraints(schema, field)
		if defaultValue != "" {
			schema["default"] = parseDefaultValue(defaultValue, field.Type)
		}

		param := map[string]any{
			"name":     name,
			"in":       "query",
			"required": strings.Contains(field.Tag.Get("binding"), "required"),
			"schema":   schema,
		}
		if baseType(field.Type).Kind() == reflect.Slice {
			param["style"] = "form"
			param["explode"] = true
		}
		params = append(params, param)
	}
	return params
}

func (p parameterSpec) build(g *generator) map[string]any {
	item := map[string]any{
		"name":     p.Name,
		"in":       p.In,
		"required": p.Required,
		"schema":   p.Schema(g),
	}
	if p.Description != "" {
		item["description"] = p.Description
	}
	return item
}

func buildOperationID(method, path string) string {
	replacer := strings.NewReplacer("/", "_", "{", "", "}", "", "-", "_", ".", "_")
	value := strings.Trim(replacer.Replace(strings.Trim(path, "/")), "_")
	if value == "" {
		value = "root"
	}
	return strings.ToLower(method) + "_" + value
}

func tag(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"description": description,
	}
}

func hasLookupParam(path string) bool {
	for _, token := range []string{"{id}", "{key}", "{md5}", "{resource_id}", "{uid}", "{project_id}", "{bucketName}", "{proxyPath}"} {
		if strings.Contains(path, token) {
			return true
		}
	}
	return false
}

func op(method, path, tag, summary string, options ...operationOption) operation {
	item := operation{
		Method:       method,
		Path:         path,
		Tag:          tag,
		Summary:      summary,
		ResponseMode: responseEnvelope,
	}
	for _, option := range options {
		option(&item)
	}
	return item
}

func withDescription(value string) operationOption {
	return func(op *operation) { op.Description = value }
}

func withSecurity(permission string) operationOption {
	return func(op *operation) {
		op.Security = true
		op.Permission = permission
	}
}

func withAuthOnly() operationOption {
	return func(op *operation) { op.Security = true }
}

func withDevOnly() operationOption {
	return func(op *operation) { op.DevOnly = true }
}

func withIdempotency() operationOption {
	return func(op *operation) { op.Idempotent = true }
}

func withRouterMethod(method string) operationOption {
	return func(op *operation) { op.RouterMethod = method }
}

func withQueryType[T any]() operationOption {
	return func(op *operation) { op.QueryType = typeOf[T]() }
}

func withParams(params ...parameterSpec) operationOption {
	return func(op *operation) { op.Parameters = append(op.Parameters, params...) }
}

func withErrors(statuses ...int) operationOption {
	return func(op *operation) { op.ErrorStatus = append(op.ErrorStatus, statuses...) }
}

func withJSONBodyType[T any]() operationOption {
	return func(op *operation) {
		op.RequestBody = typeSchema[T]()
		op.RequestContentType = "application/json"
	}
}

func withJSONBodySchema(schema schemaBuilder) operationOption {
	return func(op *operation) {
		op.RequestBody = schema
		op.RequestContentType = "application/json"
	}
}

func withRequestBodySchema(contentType string, schema schemaBuilder) operationOption {
	return func(op *operation) {
		op.RequestBody = schema
		op.RequestContentType = contentType
	}
}

func withEnvelopeType[T any]() operationOption {
	return withEnvelopeResponse(typeSchema[T]())
}

func withEnvelopeResponse(schema schemaBuilder) operationOption {
	return func(op *operation) {
		op.ResponseMode = responseEnvelope
		op.Success = schema
	}
}

func withRawJSONResponse(schema schemaBuilder) operationOption {
	return func(op *operation) {
		op.ResponseMode = responseRawJSON
		op.Success = schema
		op.SuccessContentType = "application/json"
	}
}

func withBinaryResponse(contentType string) operationOption {
	return func(op *operation) {
		op.ResponseMode = responseBinary
		op.SuccessContentType = contentType
	}
}

func withSSEResponse() operationOption {
	return func(op *operation) {
		op.ResponseMode = responseSSE
	}
}

func field(name string, schema schemaBuilder) objectField {
	return objectField{name: name, schema: schema}
}

func queryParam(name string, required bool, schema schemaBuilder, description string) parameterSpec {
	return parameterSpec{Name: name, In: "query", Required: required, Schema: schema, Description: description}
}

func pathStringParam(name, description string) parameterSpec {
	return parameterSpec{Name: name, In: "path", Required: true, Schema: stringSchema(), Description: description}
}

func pathIntParam(name, description string) parameterSpec {
	return parameterSpec{Name: name, In: "path", Required: true, Schema: int64Schema(), Description: description}
}

func objSchema(fields ...objectField) schemaBuilder {
	return func(g *generator) map[string]any {
		properties := map[string]any{}
		var required []string
		for _, item := range fields {
			properties[item.name] = item.schema(g)
			required = append(required, item.name)
		}
		return map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}
	}
}

func arraySchema(item schemaBuilder) schemaBuilder {
	return func(g *generator) map[string]any {
		return map[string]any{
			"type":  "array",
			"items": item(g),
		}
	}
}

func mapSchema(value schemaBuilder) schemaBuilder {
	return func(g *generator) map[string]any {
		schema := map[string]any{"type": "object"}
		if value == nil {
			schema["additionalProperties"] = true
			return schema
		}
		schema["additionalProperties"] = value(g)
		return schema
	}
}

func refBuilder(name string) schemaBuilder {
	return func(*generator) map[string]any { return refSchema(name) }
}

func typeSchema[T any]() schemaBuilder {
	typ := typeOf[T]()
	return func(g *generator) map[string]any {
		return g.schemaForType(typ)
	}
}

func anySchema() schemaBuilder {
	return func(*generator) map[string]any { return map[string]any{} }
}

func stringSchema() schemaBuilder {
	return func(*generator) map[string]any { return map[string]any{"type": "string"} }
}

func boolSchema() schemaBuilder {
	return func(*generator) map[string]any { return map[string]any{"type": "boolean"} }
}

func int32Schema() schemaBuilder {
	return func(*generator) map[string]any { return map[string]any{"type": "integer", "format": "int32"} }
}

func int64Schema() schemaBuilder {
	return func(*generator) map[string]any { return map[string]any{"type": "integer", "format": "int64"} }
}

func pageSchema(item schemaBuilder) schemaBuilder {
	return objSchema(
		field("data", arraySchema(item)),
		field("total", int64Schema()),
		field("page", int32Schema()),
		field("size", int32Schema()),
	)
}

func messageSchema() schemaBuilder {
	return objSchema(field("message", stringSchema()))
}

func messageWithCountSchema(fieldName string) schemaBuilder {
	return objSchema(
		field("message", stringSchema()),
		field(fieldName, int32Schema()),
	)
}

func pointsStatsSchema() schemaBuilder {
	return objSchema(
		field("points", int64Schema()),
		field("rank", int64Schema()),
		field("source_stats", mapSchema(objSchema(
			field("earned", int64Schema()),
			field("spent", int64Schema()),
		))),
	)
}

func contributionStatsSchema() schemaBuilder {
	return objSchema(
		field("total_count", int64Schema()),
		field("pending_count", int64Schema()),
		field("approved_count", int64Schema()),
		field("rejected_count", int64Schema()),
		field("total_points", int64Schema()),
	)
}

func exportConversationSchema() schemaBuilder {
	return objSchema(
		field("conversation", typeSchema[chatdto.ConversationResponse]()),
		field("messages", arraySchema(refBuilder("ChatMessage"))),
	)
}

func chatStreamRequestSchema() schemaBuilder {
	return objSchema(
		field("conversation_id", int64Schema()),
		field("message", refBuilder("ChatMessage")),
		field("checkpoint_id", stringSchema()),
		field("resume_input", stringSchema()),
	)
}

func multipartUploadSchema() schemaBuilder {
	return func(*generator) map[string]any {
		return map[string]any{
			"type": "object",
			"properties": map[string]any{
				"file": map[string]any{
					"type":   "string",
					"format": "binary",
				},
				"tags": map[string]any{
					"type":        "string",
					"description": "JSON 字符串形式的标签，例如 `{\"module\":\"docs\"}`",
				},
				"mimeType": map[string]any{
					"type":        "string",
					"description": "自定义 MIME 类型",
				},
			},
			"required": []string{"file"},
		}
	}
}

func refSchema(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (g *generator) schemaForType(t reflect.Type) map[string]any {
	if t == nil {
		return map[string]any{}
	}

	nullable := false
	for t.Kind() == reflect.Ptr {
		nullable = true
		t = t.Elem()
	}

	schema := g.nonNullableSchema(t)
	if nullable {
		return nullableSchema(schema)
	}
	return schema
}

func (g *generator) nonNullableSchema(t reflect.Type) map[string]any {
	if t == timeType {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	if isDynamicJSONType(t) {
		return map[string]any{"type": "object", "additionalProperties": true}
	}

	switch t.Kind() {
	case reflect.Struct:
		if shouldUseComponent(t) {
			name := schemaName(t)
			if _, exists := g.schemas[name]; exists {
				return refSchema(name)
			}
			g.schemas[name] = map[string]any{}
			g.schemas[name] = g.buildStructSchema(t)
			return refSchema(name)
		}
		return g.buildStructSchema(t)
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]any{"type": "integer", "format": "int32"}
	case reflect.Int64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]any{"type": "integer", "format": "int64", "minimum": 0}
	case reflect.Uint64:
		return map[string]any{"type": "integer", "format": "int64", "minimum": 0}
	case reflect.Float32:
		return map[string]any{"type": "number", "format": "float"}
	case reflect.Float64:
		return map[string]any{"type": "number", "format": "double"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{
			"type":  "array",
			"items": g.schemaForType(t.Elem()),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": g.schemaForType(t.Elem()),
		}
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (g *generator) buildStructSchema(t reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string
	g.appendStructFields(t, properties, &required)

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		sort.Strings(required)
		schema["required"] = required
	}
	return schema
}

func (g *generator) appendStructFields(t reflect.Type, properties map[string]any, required *[]string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if field.Anonymous && field.Tag.Get("json") == "" {
			embeddedType := baseType(field.Type)
			if embeddedType.Kind() == reflect.Struct {
				g.appendStructFields(embeddedType, properties, required)
				continue
			}
		}

		name, omitempty, skip := jsonFieldName(field)
		if skip || name == "" {
			continue
		}

		schema := g.schemaForType(field.Type)
		applyBindingConstraints(schema, field)
		properties[name] = schema
		if strings.Contains(field.Tag.Get("binding"), "required") && !omitempty {
			*required = append(*required, name)
		}
	}
}

func shouldUseComponent(t reflect.Type) bool {
	return t.PkgPath() != "" && t.Name() != ""
}

func isDynamicJSONType(t reflect.Type) bool {
	if t.PkgPath() == "gorm.io/datatypes" && t.Name() == "JSON" {
		return true
	}
	if t.PkgPath() == "encoding/json" && t.Name() == "RawMessage" {
		return true
	}
	return false
}

func schemaName(t reflect.Type) string {
	pkg := filepath.Base(t.PkgPath())
	if pkg == "." || pkg == string(filepath.Separator) {
		pkg = "schema"
	}
	return pkg + "_" + t.Name()
}

func nullableSchema(schema map[string]any) map[string]any {
	if _, isRef := schema["$ref"]; isRef {
		return map[string]any{
			"allOf":    []any{schema},
			"nullable": true,
		}
	}

	copied := map[string]any{}
	for key, value := range schema {
		copied[key] = value
	}
	copied["nullable"] = true
	return copied
}

func baseType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, true
	}
	if tag == "" {
		return field.Name, false, false
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = field.Name
	}
	return name, contains(parts[1:], "omitempty"), false
}

func formFieldName(field reflect.StructField) (string, string) {
	tag := field.Tag.Get("form")
	if tag == "" {
		return "", ""
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	var defaultValue string
	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "default=") {
			defaultValue = strings.TrimPrefix(part, "default=")
		}
	}
	return name, defaultValue
}

func parseDefaultValue(value string, typ reflect.Type) any {
	switch baseType(typ).Kind() {
	case reflect.Bool:
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
			return parsed
		}
	}
	return value
}

func applyBindingConstraints(schema map[string]any, field reflect.StructField) {
	rules := strings.Split(field.Tag.Get("binding"), ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" || rule == "required" || rule == "omitempty" || rule == "dive" {
			continue
		}

		switch {
		case strings.HasPrefix(rule, "min="):
			applyMinConstraint(schema, baseType(field.Type), strings.TrimPrefix(rule, "min="))
		case strings.HasPrefix(rule, "max="):
			applyMaxConstraint(schema, baseType(field.Type), strings.TrimPrefix(rule, "max="))
		case strings.HasPrefix(rule, "len="):
			value := strings.TrimPrefix(rule, "len=")
			applyMinConstraint(schema, baseType(field.Type), value)
			applyMaxConstraint(schema, baseType(field.Type), value)
		case strings.HasPrefix(rule, "oneof="):
			schema["enum"] = parseEnumValues(strings.TrimPrefix(rule, "oneof="), baseType(field.Type))
		case strings.HasPrefix(rule, "gt="):
			if parsed, err := strconv.ParseFloat(strings.TrimPrefix(rule, "gt="), 64); err == nil {
				schema["exclusiveMinimum"] = parsed
			}
		}
	}
}

func applyMinConstraint(schema map[string]any, typ reflect.Type, raw string) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return
	}
	switch typ.Kind() {
	case reflect.String:
		schema["minLength"] = int(value)
	case reflect.Slice, reflect.Array:
		schema["minItems"] = int(value)
	default:
		schema["minimum"] = value
	}
}

func applyMaxConstraint(schema map[string]any, typ reflect.Type, raw string) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return
	}
	switch typ.Kind() {
	case reflect.String:
		schema["maxLength"] = int(value)
	case reflect.Slice, reflect.Array:
		schema["maxItems"] = int(value)
	default:
		schema["maximum"] = value
	}
}

func parseEnumValues(raw string, typ reflect.Type) []any {
	var values []any
	for _, token := range strings.Fields(raw) {
		switch typ.Kind() {
		case reflect.Bool:
			if parsed, err := strconv.ParseBool(token); err == nil {
				values = append(values, parsed)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if parsed, err := strconv.ParseInt(token, 10, 64); err == nil {
				values = append(values, parsed)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if parsed, err := strconv.ParseUint(token, 10, 64); err == nil {
				values = append(values, parsed)
			}
		default:
			values = append(values, token)
		}
	}
	return values
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func buildOperations() []operation {
	ops := []operation{}

	ops = append(ops,
		op("GET", "/health", "Health", "健康检查",
			withRawJSONResponse(objSchema(
				field("status", stringSchema()),
				field("message", stringSchema()),
			)),
		),
		op("POST", "/api/mcp", "MCP", "MCP HTTP 入口",
			withAuthOnly(),
			withRouterMethod("ANY"),
			withDescription("Gin 路由使用 `Any(\"/api/mcp\")`，本规范用 POST 代表该入口。具体 JSON-RPC / Streamable HTTP 细节由 `mcp-go` 实现控制。"),
			withRequestBodySchema("application/json", mapSchema(anySchema())),
			withRawJSONResponse(mapSchema(anySchema())),
		),
		op("POST", "/api/v0/auth/wechat-login", "Auth", "微信登录",
			withJSONBodyType[req.WechatLoginRequest](),
			withEnvelopeType[resp.WechatLoginResponse](),
			withErrors(401),
		),
		op("POST", "/api/v0/auth/refresh", "Auth", "刷新访问令牌",
			withJSONBodyType[req.RefreshTokenRequest](),
			withEnvelopeType[resp.WechatLoginResponse](),
			withErrors(401),
		),
		op("POST", "/api/v0/auth/mock-wechat-login", "Auth", "模拟微信登录",
			withDescription("仅在非 release 模式注册，用于 E2E 测试与本地联调。"),
			withDevOnly(),
			withJSONBodyType[req.MockWechatLoginRequest](),
			withEnvelopeType[resp.WechatLoginResponse](),
		),
		op("POST", "/api/v0/admin/auth/login", "Auth", "管理界面手机号密码登录",
			withJSONBodyType[req.AdminLoginRequest](),
			withEnvelopeType[resp.WechatLoginResponse](),
			withErrors(400, 401),
		),
		op("GET", "/api/v0/reviews/teacher", "Reviews", "按教师查询评价",
			withParams(
				queryParam("teacher_name", true, stringSchema(), "教师姓名"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[models.TeacherReview]())),
		),
		op("GET", "/api/v0/config/{key}", "Config", "按键读取配置",
			withParams(pathStringParam("key", "配置键")),
			withEnvelopeType[resp.ConfigResponse](),
		),
		op("GET", "/api/v0/heroes/", "Heroes", "获取英雄榜",
			withEnvelopeResponse(arraySchema(stringSchema())),
		),
		op("POST", "/api/v0/auth/logout", "Auth", "退出当前设备登录",
			withAuthOnly(),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/auth/logout-all", "Auth", "退出全部设备登录",
			withAuthOnly(),
			withEnvelopeResponse(messageWithCountSchema("deleted_session_count")),
		),
		op("GET", "/api/v0/user/profile", "User", "获取当前用户资料",
			withSecurity(constant.PermissionUserGet),
			withEnvelopeType[resp.UserProfileResponse](),
		),
		op("PUT", "/api/v0/user/profile", "User", "更新当前用户资料",
			withSecurity(constant.PermissionUserUpdate),
			withJSONBodyType[req.UpdateProfileRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/user/features", "User", "获取当前用户功能列表",
			withSecurity(constant.PermissionUserGet),
			withEnvelopeType[resp.UserFeaturesResponse](),
		),
		op("GET", "/api/v0/user/login-days", "User", "获取过去 100 天登录天数",
			withSecurity(constant.PermissionUserGet),
			withEnvelopeResponse(objSchema(
				field("past_days", int32Schema()),
				field("login_days", int32Schema()),
			)),
		),
		op("POST", "/api/v0/oss/token", "Storage", "生成 OSS/CDN 签名",
			withSecurity(constant.PermissionOSSTokenGet),
			withJSONBodyType[req.OSSGetTokenRequest](),
			withEnvelopeType[resp.OSSGetTokenResponse](),
		),
		op("POST", "/api/v0/reviews/", "Reviews", "创建教师评价",
			withSecurity(constant.PermissionReviewCreate),
			withIdempotency(),
			withJSONBodyType[req.CreateReviewRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/reviews/user", "Reviews", "获取当前用户评价记录",
			withSecurity(constant.PermissionReviewGetSelf),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[models.TeacherReview]())),
		),
		op("GET", "/api/v0/reviews/", "Reviews", "管理员查询评价列表",
			withSecurity(constant.PermissionReviewManage),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("size", false, int32Schema(), "每页数量"),
				queryParam("teacher_name", false, stringSchema(), "教师姓名"),
				queryParam("status", false, int32Schema(), "审核状态"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[models.TeacherReview]())),
		),
		op("POST", "/api/v0/reviews/{id}/approve", "Reviews", "审核通过评价",
			withSecurity(constant.PermissionReviewManage),
			withIdempotency(),
			withParams(pathIntParam("id", "评价 ID")),
			withJSONBodySchema(objSchema(field("admin_note", stringSchema()))),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/reviews/{id}/reject", "Reviews", "审核拒绝评价",
			withSecurity(constant.PermissionReviewManage),
			withIdempotency(),
			withParams(pathIntParam("id", "评价 ID")),
			withJSONBodySchema(objSchema(field("admin_note", stringSchema()))),
			withEnvelopeResponse(messageSchema()),
		),
		op("DELETE", "/api/v0/reviews/{id}", "Reviews", "删除评价",
			withSecurity(constant.PermissionReviewManage),
			withParams(pathIntParam("id", "评价 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/coursetable/", "CourseTable", "获取用户课程表",
			withSecurity(constant.PermissionCourseTableGet),
			withQueryType[req.GetCourseTableRequest](),
			withEnvelopeType[resp.CourseTableResponse](),
		),
		op("GET", "/api/v0/coursetable/search", "CourseTable", "搜索班级",
			withSecurity(constant.PermissionCourseTableClassSearch),
			withQueryType[req.SearchClassRequest](),
			withEnvelopeType[resp.SearchClassResponse](),
		),
		op("GET", "/api/v0/coursetable/bind-count", "CourseTable", "获取课表绑定次数",
			withSecurity(constant.PermissionCourseTableGet),
			withEnvelopeResponse(objSchema(field("bind_count", int32Schema()))),
		),
		op("PUT", "/api/v0/coursetable/class", "CourseTable", "更新用户班级",
			withSecurity(constant.PermissionCourseTableClassUpdate),
			withJSONBodyType[req.UpdateUserClassRequest](),
			withEnvelopeResponse(stringSchema()),
		),
		op("PUT", "/api/v0/coursetable/", "CourseTable", "编辑个人课表单元格",
			withSecurity(constant.PermissionCourseTableUpdate),
			withJSONBodyType[req.EditCourseCellRequest](),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/coursetable/schedule", "CourseTable", "重置个人课表",
			withSecurity(constant.PermissionCourseTableUpdate),
			withParams(queryParam("semester", true, stringSchema(), "学期")),
			withEnvelopeResponse(stringSchema()),
		),
		op("POST", "/api/v0/coursetable/reset/{id}", "CourseTable", "重置指定用户课表绑定次数",
			withSecurity(constant.PermissionCourseTableManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/admin/coursetables", "AdminCourseTables", "管理员分页查询课表",
			withSecurity(constant.PermissionCourseTableManage),
			withParams(
				queryParam("class_id", false, stringSchema(), "班级 ID"),
				queryParam("semester", false, stringSchema(), "学期"),
				queryParam("keyword", false, stringSchema(), "关键字"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[models.CourseTable]())),
		),
		op("GET", "/api/v0/admin/coursetables/{id}", "AdminCourseTables", "管理员获取课表详情",
			withSecurity(constant.PermissionCourseTableManage),
			withParams(pathIntParam("id", "课表 ID")),
			withEnvelopeType[models.CourseTable](),
		),
		op("POST", "/api/v0/admin/coursetables", "AdminCourseTables", "管理员创建课表",
			withSecurity(constant.PermissionCourseTableManage),
			withJSONBodySchema(objSchema(
				field("class_id", stringSchema()),
				field("semester", stringSchema()),
				field("course_data", anySchema()),
			)),
			withEnvelopeType[models.CourseTable](),
		),
		op("PUT", "/api/v0/admin/coursetables/{id}", "AdminCourseTables", "管理员更新课表",
			withSecurity(constant.PermissionCourseTableManage),
			withParams(pathIntParam("id", "课表 ID")),
			withJSONBodySchema(objSchema(
				field("class_id", stringSchema()),
				field("semester", stringSchema()),
				field("course_data", anySchema()),
			)),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/admin/coursetables/{id}", "AdminCourseTables", "管理员删除课表",
			withSecurity(constant.PermissionCourseTableManage),
			withParams(pathIntParam("id", "课表 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/failrate/search", "FailRate", "搜索挂科率",
			withSecurity(constant.PermissionFailRate),
			withQueryType[req.SearchFailRateRequest](),
			withEnvelopeType[resp.FailRateListResponse](),
		),
		op("GET", "/api/v0/failrate/rand", "FailRate", "随机返回挂科率",
			withSecurity(constant.PermissionFailRate),
			withEnvelopeType[resp.FailRateListResponse](),
		),
		op("GET", "/api/v0/admin/failrates", "AdminFailRates", "管理员分页查询挂科率",
			withSecurity(constant.PermissionFailRateManage),
			withParams(
				queryParam("keyword", false, stringSchema(), "课程关键词"),
				queryParam("department", false, stringSchema(), "开课单位"),
				queryParam("semester", false, stringSchema(), "学期"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[models.FailRate]())),
		),
		op("GET", "/api/v0/admin/failrates/{id}", "AdminFailRates", "管理员获取挂科率详情",
			withSecurity(constant.PermissionFailRateManage),
			withParams(pathIntParam("id", "挂科率 ID")),
			withEnvelopeType[models.FailRate](),
		),
		op("POST", "/api/v0/admin/failrates", "AdminFailRates", "管理员创建挂科率",
			withSecurity(constant.PermissionFailRateManage),
			withJSONBodySchema(objSchema(
				field("course_name", stringSchema()),
				field("department", stringSchema()),
				field("semester", stringSchema()),
				field("failrate", anySchema()),
			)),
			withEnvelopeType[models.FailRate](),
		),
		op("PUT", "/api/v0/admin/failrates/{id}", "AdminFailRates", "管理员更新挂科率",
			withSecurity(constant.PermissionFailRateManage),
			withParams(pathIntParam("id", "挂科率 ID")),
			withJSONBodySchema(objSchema(
				field("course_name", stringSchema()),
				field("department", stringSchema()),
				field("semester", stringSchema()),
				field("failrate", anySchema()),
			)),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/admin/failrates/{id}", "AdminFailRates", "管理员删除挂科率",
			withSecurity(constant.PermissionFailRateManage),
			withParams(pathIntParam("id", "挂科率 ID")),
			withEnvelopeResponse(stringSchema()),
		),
	)

	ops = append(ops,
		op("POST", "/api/v0/heroes/", "Heroes", "创建英雄",
			withSecurity(constant.PermissionHeroManage),
			withIdempotency(),
			withJSONBodyType[req.CreateHeroRequest](),
			withEnvelopeType[models.Hero](),
		),
		op("PUT", "/api/v0/heroes/{id}", "Heroes", "更新英雄",
			withSecurity(constant.PermissionHeroManage),
			withParams(pathIntParam("id", "英雄 ID")),
			withJSONBodyType[req.UpdateHeroRequest](),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/heroes/{id}", "Heroes", "删除英雄",
			withSecurity(constant.PermissionHeroManage),
			withParams(pathIntParam("id", "英雄 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/heroes/search", "Heroes", "搜索英雄",
			withSecurity(constant.PermissionHeroManage),
			withQueryType[req.SearchHeroRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[models.Hero]())),
		),
		op("POST", "/api/v0/config/", "Config", "创建配置项",
			withSecurity(constant.PermissionConfigManage),
			withIdempotency(),
			withJSONBodyType[req.CreateConfigRequest](),
			withEnvelopeType[models.SystemConfig](),
		),
		op("PUT", "/api/v0/config/{key}", "Config", "更新配置项",
			withSecurity(constant.PermissionConfigManage),
			withIdempotency(),
			withParams(pathStringParam("key", "配置键")),
			withJSONBodyType[req.UpdateConfigRequest](),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/config/{key}", "Config", "删除配置项",
			withSecurity(constant.PermissionConfigManage),
			withIdempotency(),
			withParams(pathStringParam("key", "配置键")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/config/search", "Config", "搜索配置项",
			withSecurity(constant.PermissionConfigManage),
			withQueryType[req.SearchConfigRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[models.SystemConfig]())),
		),
		op("GET", "/api/v0/store/{resource_id}/url", "Storage", "获取文件访问地址",
			withAuthOnly(),
			withParams(pathStringParam("resource_id", "资源 ID")),
			withQueryType[req.GetFileURLRequest](),
			withEnvelopeType[resp.GetFileURLResponse](),
		),
		op("GET", "/api/v0/store/{resource_id}/stream", "Storage", "获取文件流",
			withAuthOnly(),
			withParams(pathStringParam("resource_id", "资源 ID")),
			withBinaryResponse("application/octet-stream"),
		),
		op("POST", "/api/v0/store", "Storage", "上传文件",
			withSecurity(constant.PermissionS3Manage),
			withRequestBodySchema("multipart/form-data", multipartUploadSchema()),
			withEnvelopeType[resp.UploadFileResponse](),
		),
		op("DELETE", "/api/v0/store/{resource_id}", "Storage", "删除文件",
			withSecurity(constant.PermissionS3Manage),
			withParams(pathStringParam("resource_id", "资源 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/store/list", "Storage", "获取文件列表",
			withSecurity(constant.PermissionS3Manage),
			withEnvelopeResponse(arraySchema(typeSchema[models.S3Data]())),
		),
		op("GET", "/api/v0/store/expired", "Storage", "获取过期文件列表",
			withSecurity(constant.PermissionS3Manage),
			withEnvelopeResponse(arraySchema(typeSchema[models.S3Resource]())),
		),
		op("GET", "/api/v0/points/", "Points", "获取用户积分",
			withSecurity(constant.PermissionPointGet),
			withParams(queryParam("user_id", false, int64Schema(), "目标用户 ID，普通用户只能查询自己")),
			withEnvelopeType[resp.UserPointsResponse](),
		),
		op("GET", "/api/v0/points/transactions", "Points", "获取积分交易记录",
			withSecurity(constant.PermissionPointGet),
			withQueryType[req.GetPointsTransactionsRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.PointsTransactionResponse]())),
		),
		op("POST", "/api/v0/points/spend", "Points", "消费积分",
			withSecurity(constant.PermissionPointSpend),
			withIdempotency(),
			withJSONBodyType[req.SpendPointsRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/points/stats", "Points", "获取积分统计",
			withSecurity(constant.PermissionPointGet),
			withParams(queryParam("user_id", false, int64Schema(), "目标用户 ID，普通用户只能查询自己")),
			withEnvelopeResponse(pointsStatsSchema()),
		),
		op("GET", "/api/v0/gpa/backup", "GPA", "获取当前用户绩点备份列表",
			withSecurity(constant.PermissionUserGet),
			withEnvelopeResponse(arraySchema(typeSchema[resp.GPABackupResponse]())),
		),
		op("GET", "/api/v0/gpa/backup/{id}", "GPA", "获取当前用户单个绩点备份",
			withSecurity(constant.PermissionUserGet),
			withParams(pathIntParam("id", "备份 ID")),
			withEnvelopeType[resp.GPABackupResponse](),
		),
		op("POST", "/api/v0/gpa/backup", "GPA", "创建当前用户绩点备份",
			withSecurity(constant.PermissionUserUpdate),
			withIdempotency(),
			withRequestBodySchema("application/json", anySchema()),
			withEnvelopeType[resp.GPABackupResponse](),
		),
		op("DELETE", "/api/v0/gpa/backup/{id}", "GPA", "删除当前用户绩点备份",
			withSecurity(constant.PermissionUserUpdate),
			withParams(pathIntParam("id", "备份 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/points/grant", "Points", "管理员手动增减积分",
			withSecurity(constant.PermissionPointManage),
			withIdempotency(),
			withJSONBodyType[req.GrantPointsRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/contributions/", "Contributions", "创建投稿",
			withSecurity(constant.PermissionContributionCreate),
			withIdempotency(),
			withJSONBodyType[req.CreateContributionRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/contributions/", "Contributions", "获取投稿列表",
			withSecurity(constant.PermissionContributionGet),
			withQueryType[req.GetContributionsRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.ContributionResponse]())),
		),
		op("GET", "/api/v0/contributions/{id}", "Contributions", "获取投稿详情",
			withSecurity(constant.PermissionContributionGet),
			withParams(pathIntParam("id", "投稿 ID")),
			withEnvelopeType[resp.ContributionResponse](),
		),
		op("GET", "/api/v0/contributions/stats", "Contributions", "获取当前用户投稿统计",
			withSecurity(constant.PermissionContributionGet),
			withEnvelopeResponse(contributionStatsSchema()),
		),
		op("POST", "/api/v0/contributions/{id}/review", "Contributions", "审核投稿",
			withSecurity(constant.PermissionContributionManage),
			withIdempotency(),
			withParams(pathIntParam("id", "投稿 ID")),
			withJSONBodyType[req.ReviewContributionRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/contributions/stats-admin", "Contributions", "获取全站投稿统计",
			withSecurity(constant.PermissionContributionManage),
			withEnvelopeType[resp.AdminContributionStatsResponse](),
		),
	)

	ops = append(ops,
		op("POST", "/api/v0/countdowns/", "Countdowns", "创建倒数日",
			withSecurity(constant.PermissionCountdown),
			withIdempotency(),
			withJSONBodyType[req.CreateCountdownRequest](),
			withEnvelopeType[resp.CountdownResponse](),
		),
		op("GET", "/api/v0/countdowns/", "Countdowns", "获取倒数日列表",
			withSecurity(constant.PermissionCountdown),
			withEnvelopeResponse(arraySchema(typeSchema[resp.CountdownResponse]())),
		),
		op("GET", "/api/v0/countdowns/{id}", "Countdowns", "获取倒数日详情",
			withSecurity(constant.PermissionCountdown),
			withParams(pathIntParam("id", "倒数日 ID")),
			withEnvelopeType[resp.CountdownResponse](),
		),
		op("PUT", "/api/v0/countdowns/{id}", "Countdowns", "更新倒数日",
			withSecurity(constant.PermissionCountdown),
			withIdempotency(),
			withParams(pathIntParam("id", "倒数日 ID")),
			withJSONBodyType[req.UpdateCountdownRequest](),
			withEnvelopeType[resp.CountdownResponse](),
		),
		op("DELETE", "/api/v0/countdowns/{id}", "Countdowns", "删除倒数日",
			withSecurity(constant.PermissionCountdown),
			withParams(pathIntParam("id", "倒数日 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/study-tasks/", "StudyTasks", "创建学习任务",
			withSecurity(constant.PermissionStudyTask),
			withIdempotency(),
			withJSONBodyType[req.CreateStudyTaskRequest](),
			withEnvelopeType[resp.StudyTaskResponse](),
		),
		op("GET", "/api/v0/study-tasks/", "StudyTasks", "获取学习任务列表",
			withSecurity(constant.PermissionStudyTask),
			withQueryType[req.GetStudyTasksRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.StudyTaskResponse]())),
		),
		op("GET", "/api/v0/study-tasks/{id}", "StudyTasks", "获取学习任务详情",
			withSecurity(constant.PermissionStudyTask),
			withParams(pathIntParam("id", "任务 ID")),
			withEnvelopeType[resp.StudyTaskResponse](),
		),
		op("PUT", "/api/v0/study-tasks/{id}", "StudyTasks", "更新学习任务",
			withSecurity(constant.PermissionStudyTask),
			withIdempotency(),
			withParams(pathIntParam("id", "任务 ID")),
			withJSONBodyType[req.UpdateStudyTaskRequest](),
			withEnvelopeType[resp.StudyTaskResponse](),
		),
		op("DELETE", "/api/v0/study-tasks/{id}", "StudyTasks", "删除学习任务",
			withSecurity(constant.PermissionStudyTask),
			withParams(pathIntParam("id", "任务 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/study-tasks/stats", "StudyTasks", "获取学习任务统计",
			withSecurity(constant.PermissionStudyTask),
			withEnvelopeType[resp.StudyTaskStatsResponse](),
		),
		op("GET", "/api/v0/study-tasks/completed", "StudyTasks", "获取已完成学习任务",
			withSecurity(constant.PermissionStudyTask),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[resp.StudyTaskResponse]())),
		),
		op("GET", "/api/v0/materials/", "Materials", "获取资料列表",
			withSecurity(constant.PermissionMaterialGet),
			withQueryType[req.MaterialListRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.MaterialListResponse]())),
		),
		op("GET", "/api/v0/materials/top", "Materials", "获取热门资料",
			withSecurity(constant.PermissionMaterialGet),
			withQueryType[req.TopMaterialsRequest](),
			withEnvelopeResponse(arraySchema(typeSchema[resp.MaterialListResponse]())),
		),
		op("GET", "/api/v0/materials/hot-words", "Materials", "获取热词",
			withSecurity(constant.PermissionMaterialGet),
			withQueryType[req.HotWordsRequest](),
			withEnvelopeResponse(arraySchema(typeSchema[resp.HotWordsResponse]())),
		),
		op("GET", "/api/v0/materials/search", "Materials", "搜索资料",
			withSecurity(constant.PermissionMaterialGet),
			withQueryType[req.MaterialSearchRequest](),
			withEnvelopeType[resp.MaterialSearchResponse](),
		),
		op("GET", "/api/v0/materials/{md5}", "Materials", "获取资料详情",
			withSecurity(constant.PermissionMaterialGet),
			withParams(pathStringParam("md5", "资料 MD5")),
			withEnvelopeType[resp.MaterialDetailResponse](),
		),
		op("POST", "/api/v0/materials/{md5}/rating", "Materials", "资料评分",
			withSecurity(constant.PermissionMaterialRate),
			withParams(pathStringParam("md5", "资料 MD5")),
			withJSONBodyType[req.MaterialRatingRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/materials/{md5}/download", "Materials", "记录资料下载",
			withSecurity(constant.PermissionMaterialDownload),
			withParams(pathStringParam("md5", "资料 MD5")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/material-categories/", "Materials", "获取资料分类",
			withSecurity(constant.PermissionMaterialCategoryGet),
			withQueryType[req.MaterialCategoryListRequest](),
			withEnvelopeResponse(arraySchema(typeSchema[resp.MaterialCategoryResponse]())),
		),
		op("GET", "/api/v0/notifications/", "Notifications", "获取通知列表",
			withSecurity(constant.PermissionNotificationGet),
			withQueryType[req.GetNotificationsRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.NotificationSimpleResponse]())),
		),
		op("GET", "/api/v0/notifications/{id}", "Notifications", "获取通知详情",
			withSecurity(constant.PermissionNotificationGet),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeType[resp.NotificationResponse](),
		),
		op("GET", "/api/v0/categories/", "Notifications", "获取通知分类",
			withSecurity(constant.PermissionNotificationGet),
			withEnvelopeResponse(arraySchema(typeSchema[resp.NotificationCategoryResponse]())),
		),
		op("GET", "/api/v0/questions/projects", "Questions", "获取题库项目列表",
			withSecurity(constant.PermissionQuestion),
			withEnvelopeResponse(arraySchema(typeSchema[resp.QuestionProjectResponse]())),
		),
		op("GET", "/api/v0/questions/list", "Questions", "获取题目 ID 列表",
			withSecurity(constant.PermissionQuestion),
			withQueryType[req.GetQuestionRequest](),
			withEnvelopeType[resp.QuestionListResponse](),
		),
		op("GET", "/api/v0/questions/{id}", "Questions", "获取题目详情",
			withSecurity(constant.PermissionQuestion),
			withParams(pathIntParam("id", "题目 ID")),
			withEnvelopeType[resp.QuestionResponse](),
		),
		op("POST", "/api/v0/questions/study", "Questions", "记录学习次数",
			withSecurity(constant.PermissionQuestion),
			withJSONBodyType[req.RecordStudyRequest](),
			withEnvelopeResponse(nil),
		),
		op("POST", "/api/v0/questions/practice", "Questions", "记录练习次数",
			withSecurity(constant.PermissionQuestion),
			withJSONBodyType[req.SubmitPracticeRequest](),
			withEnvelopeResponse(nil),
		),
		op("GET", "/api/v0/admin/questions/projects", "AdminQuestions", "管理员分页查询题库项目",
			withSecurity(constant.PermissionQuestionProjectManage),
			withParams(
				queryParam("keyword", false, stringSchema(), "关键字"),
				queryParam("is_active", false, boolSchema(), "是否启用"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(objSchema(
				field("id", int64Schema()),
				field("name", stringSchema()),
				field("description", stringSchema()),
				field("version", int32Schema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
				field("created_at", stringSchema()),
				field("updated_at", stringSchema()),
			))),
		),
		op("GET", "/api/v0/admin/questions/projects/{id}", "AdminQuestions", "管理员获取题库项目详情",
			withSecurity(constant.PermissionQuestionProjectManage),
			withParams(pathIntParam("id", "项目 ID")),
			withEnvelopeResponse(anySchema()),
		),
		op("POST", "/api/v0/admin/questions/projects", "AdminQuestions", "管理员创建题库项目",
			withSecurity(constant.PermissionQuestionProjectManage),
			withJSONBodySchema(objSchema(
				field("name", stringSchema()),
				field("description", stringSchema()),
				field("version", int32Schema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
			)),
			withEnvelopeResponse(anySchema()),
		),
		op("PUT", "/api/v0/admin/questions/projects/{id}", "AdminQuestions", "管理员更新题库项目",
			withSecurity(constant.PermissionQuestionProjectManage),
			withParams(pathIntParam("id", "项目 ID")),
			withJSONBodySchema(objSchema(
				field("name", stringSchema()),
				field("description", stringSchema()),
				field("version", int32Schema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
			)),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/admin/questions/projects/{id}", "AdminQuestions", "管理员删除题库项目",
			withSecurity(constant.PermissionQuestionProjectManage),
			withParams(pathIntParam("id", "项目 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/admin/questions", "AdminQuestions", "管理员分页搜索题目",
			withSecurity(constant.PermissionQuestionManage),
			withParams(
				queryParam("project_id", false, int64Schema(), "项目 ID"),
				queryParam("keyword", false, stringSchema(), "标题关键字"),
				queryParam("is_active", false, boolSchema(), "是否启用"),
				queryParam("parent_id", false, int64Schema(), "父题目 ID"),
				queryParam("type", false, int32Schema(), "题目类型"),
				queryParam("sort_min", false, int32Schema(), "排序最小值"),
				queryParam("sort_max", false, int32Schema(), "排序最大值"),
				queryParam("created_from", false, stringSchema(), "创建开始时间"),
				queryParam("created_to", false, stringSchema(), "创建结束时间"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(objSchema(
				field("id", int64Schema()),
				field("project_id", int64Schema()),
				field("parent_id", int64Schema()),
				field("type", int32Schema()),
				field("title", stringSchema()),
				field("options", arraySchema(stringSchema())),
				field("answer", stringSchema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
				field("created_at", stringSchema()),
				field("updated_at", stringSchema()),
			))),
		),
		op("GET", "/api/v0/admin/questions/{id}", "AdminQuestions", "管理员获取题目详情",
			withSecurity(constant.PermissionQuestionManage),
			withParams(pathIntParam("id", "题目 ID")),
			withEnvelopeResponse(anySchema()),
		),
		op("POST", "/api/v0/admin/questions", "AdminQuestions", "管理员创建题目",
			withSecurity(constant.PermissionQuestionManage),
			withJSONBodySchema(objSchema(
				field("project_id", int64Schema()),
				field("parent_id", int64Schema()),
				field("type", int32Schema()),
				field("title", stringSchema()),
				field("options", arraySchema(stringSchema())),
				field("answer", stringSchema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
			)),
			withEnvelopeResponse(anySchema()),
		),
		op("PUT", "/api/v0/admin/questions/{id}", "AdminQuestions", "管理员更新题目",
			withSecurity(constant.PermissionQuestionManage),
			withParams(pathIntParam("id", "题目 ID")),
			withJSONBodySchema(objSchema(
				field("project_id", int64Schema()),
				field("parent_id", int64Schema()),
				field("type", int32Schema()),
				field("title", stringSchema()),
				field("options", arraySchema(stringSchema())),
				field("answer", stringSchema()),
				field("sort", int32Schema()),
				field("is_active", boolSchema()),
			)),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/admin/questions/{id}", "AdminQuestions", "管理员删除题目",
			withSecurity(constant.PermissionQuestionManage),
			withParams(pathIntParam("id", "题目 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("POST", "/api/v0/pomodoro/increment", "Pomodoro", "增加番茄钟次数",
			withSecurity(constant.PermissionPomodoro),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/pomodoro/ranking", "Pomodoro", "获取番茄钟排名",
			withSecurity(constant.PermissionPomodoro),
			withEnvelopeResponse(arraySchema(typeSchema[resp.PomodoroRankingItem]())),
		),
	)

	ops = append(ops,
		op("POST", "/api/v0/chat/conversations", "Chat", "创建对话",
			withSecurity(constant.PermissionChatStudy),
			withIdempotency(),
			withJSONBodyType[chatdto.CreateConversationRequest](),
			withEnvelopeType[chatdto.ConversationResponse](),
		),
		op("GET", "/api/v0/chat/conversations", "Chat", "获取对话列表",
			withSecurity(constant.PermissionChatStudy),
			withQueryType[chatdto.ListConversationsRequest](),
			withEnvelopeType[chatdto.ConversationListResponse](),
		),
		op("GET", "/api/v0/chat/conversations/{id}", "Chat", "获取对话历史消息",
			withSecurity(constant.PermissionChatStudy),
			withParams(pathIntParam("id", "对话 ID")),
			withEnvelopeResponse(arraySchema(refBuilder("ChatMessage"))),
		),
		op("PUT", "/api/v0/chat/conversations/{id}", "Chat", "更新对话标题",
			withSecurity(constant.PermissionChatStudy),
			withParams(pathIntParam("id", "对话 ID")),
			withJSONBodyType[chatdto.UpdateConversationRequest](),
			withEnvelopeResponse(stringSchema()),
		),
		op("DELETE", "/api/v0/chat/conversations/{id}", "Chat", "删除对话",
			withSecurity(constant.PermissionChatStudy),
			withParams(pathIntParam("id", "对话 ID")),
			withEnvelopeResponse(stringSchema()),
		),
		op("GET", "/api/v0/chat/conversations/{id}/export", "Chat", "导出对话",
			withSecurity(constant.PermissionChatStudy),
			withParams(pathIntParam("id", "对话 ID")),
			withEnvelopeResponse(exportConversationSchema()),
		),
		op("POST", "/api/v0/chat/conversation", "Chat", "发起流式对话",
			withSecurity(constant.PermissionChatStudy),
			withRequestBodySchema("application/json", chatStreamRequestSchema()),
			withSSEResponse(),
		),
		op("GET", "/api/v0/stat/system/online", "Stats", "获取系统在线人数",
			withSecurity(constant.PermissionStatisticGet),
			withEnvelopeType[resp.SystemOnlineStatResponse](),
		),
		op("GET", "/api/v0/stat/project/{project_id}/online", "Stats", "获取项目在线人数",
			withSecurity(constant.PermissionStatisticGet),
			withParams(pathIntParam("project_id", "项目 ID")),
			withEnvelopeType[resp.ProjectOnlineStatResponse](),
		),
		op("GET", "/api/v0/admin/stats/countdowns/by-user", "AdminStats", "管理员按用户统计倒数日数量",
			withSecurity(constant.PermissionStatisticManage),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(objSchema(
				field("user_id", int64Schema()),
				field("count", int64Schema()),
			))),
		),
		op("GET", "/api/v0/admin/stats/studytasks/by-user", "AdminStats", "管理员按用户统计学习清单数量",
			withSecurity(constant.PermissionStatisticManage),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(objSchema(
				field("user_id", int64Schema()),
				field("count", int64Schema()),
			))),
		),
		op("GET", "/api/v0/admin/stats/gpa-backups/by-user", "AdminStats", "管理员按用户统计绩点备份数量",
			withSecurity(constant.PermissionStatisticManage),
			withParams(
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(objSchema(
				field("user_id", int64Schema()),
				field("count", int64Schema()),
			))),
		),
		op("GET", "/api/v0/dictionary/word", "Dictionary", "随机获取词典条目",
			withSecurity(constant.PermissionDictionary),
			withEnvelopeType[models.Dictionary](),
		),
		op("GET", "/api/v0/admin/notifications/", "Notifications", "管理员获取通知列表",
			withSecurity(constant.PermissionNotificationGetAdmin),
			withQueryType[req.GetNotificationsRequest](),
			withEnvelopeResponse(pageSchema(typeSchema[resp.NotificationSimpleResponse]())),
		),
		op("GET", "/api/v0/admin/notifications/stats", "Notifications", "获取通知统计",
			withSecurity(constant.PermissionNotificationGetAdmin),
			withEnvelopeType[resp.NotificationStatsResponse](),
		),
		op("GET", "/api/v0/admin/notifications/{id}", "Notifications", "管理员获取通知详情",
			withSecurity(constant.PermissionNotificationGetAdmin),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeType[resp.NotificationResponse](),
		),
		op("POST", "/api/v0/admin/notifications/", "Notifications", "创建通知",
			withSecurity(constant.PermissionNotificationCreate),
			withIdempotency(),
			withJSONBodyType[req.CreateNotificationRequest](),
			withEnvelopeType[resp.NotificationResponse](),
		),
		op("POST", "/api/v0/admin/notifications/{id}/publish", "Notifications", "发布通知",
			withSecurity(constant.PermissionNotificationPublish),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("PUT", "/api/v0/admin/notifications/{id}", "Notifications", "更新通知",
			withSecurity(constant.PermissionNotificationUpdate),
			withParams(pathIntParam("id", "通知 ID")),
			withJSONBodyType[req.UpdateNotificationRequest](),
			withEnvelopeType[resp.NotificationResponse](),
		),
		op("POST", "/api/v0/admin/notifications/{id}/approve", "Notifications", "审核通知",
			withSecurity(constant.PermissionNotificationApprove),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withJSONBodyType[req.ApproveNotificationRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/notifications/{id}/schedule", "Notifications", "转换通知为日程",
			withSecurity(constant.PermissionNotificationSchedule),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withJSONBodyType[req.ConvertToScheduleRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("DELETE", "/api/v0/admin/notifications/{id}", "Notifications", "删除通知",
			withSecurity(constant.PermissionNotificationDelete),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/notifications/{id}/publish-admin", "Notifications", "管理员直接发布通知",
			withSecurity(constant.PermissionNotificationPublishAdmin),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/notifications/{id}/pin", "Notifications", "置顶通知",
			withSecurity(constant.PermissionNotificationPin),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/notifications/{id}/unpin", "Notifications", "取消置顶通知",
			withSecurity(constant.PermissionNotificationPin),
			withIdempotency(),
			withParams(pathIntParam("id", "通知 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/categories/", "Notifications", "创建通知分类",
			withSecurity(constant.PermissionNotificationCategoryManage),
			withIdempotency(),
			withJSONBodyType[req.CreateCategoryRequest](),
			withEnvelopeType[resp.NotificationCategoryResponse](),
		),
		op("PUT", "/api/v0/admin/categories/{id}", "Notifications", "更新通知分类",
			withSecurity(constant.PermissionNotificationCategoryManage),
			withParams(pathIntParam("id", "分类 ID")),
			withJSONBodyType[req.UpdateCategoryRequest](),
			withEnvelopeType[resp.NotificationCategoryResponse](),
		),
	)

	ops = append(ops,
		op("GET", "/api/v0/admin/features", "Features", "获取全部功能",
			withSecurity(constant.PermissionFeatureManage),
			withEnvelopeResponse(arraySchema(typeSchema[models.Feature]())),
		),
		op("GET", "/api/v0/admin/features/{key}", "Features", "获取功能详情",
			withSecurity(constant.PermissionFeatureManage),
			withParams(pathStringParam("key", "功能标识")),
			withEnvelopeType[models.Feature](),
		),
		op("POST", "/api/v0/admin/features", "Features", "创建功能",
			withSecurity(constant.PermissionFeatureManage),
			withIdempotency(),
			withJSONBodyType[req.CreateFeatureRequest](),
			withEnvelopeType[models.Feature](),
		),
		op("PUT", "/api/v0/admin/features/{key}", "Features", "更新功能",
			withSecurity(constant.PermissionFeatureManage),
			withParams(pathStringParam("key", "功能标识")),
			withJSONBodyType[req.UpdateFeatureRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("DELETE", "/api/v0/admin/features/{key}", "Features", "删除功能",
			withSecurity(constant.PermissionFeatureManage),
			withParams(pathStringParam("key", "功能标识")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/admin/features/{key}/whitelist", "Features", "获取功能白名单",
			withSecurity(constant.PermissionFeatureManage),
			withParams(
				pathStringParam("key", "功能标识"),
				queryParam("page", false, int32Schema(), "页码"),
				queryParam("page_size", false, int32Schema(), "每页数量"),
			),
			withEnvelopeResponse(pageSchema(typeSchema[resp.WhitelistUserInfo]())),
		),
		op("POST", "/api/v0/admin/features/{key}/whitelist", "Features", "授予功能权限",
			withSecurity(constant.PermissionFeatureManage),
			withIdempotency(),
			withParams(pathStringParam("key", "功能标识")),
			withJSONBodyType[req.GrantFeatureRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("POST", "/api/v0/admin/features/{key}/whitelist/batch", "Features", "批量授予功能权限",
			withSecurity(constant.PermissionFeatureManage),
			withIdempotency(),
			withParams(pathStringParam("key", "功能标识")),
			withJSONBodyType[req.BatchGrantFeatureRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("DELETE", "/api/v0/admin/features/{key}/whitelist/{uid}", "Features", "撤销功能权限",
			withSecurity(constant.PermissionFeatureManage),
			withParams(
				pathStringParam("key", "功能标识"),
				pathIntParam("uid", "用户 ID"),
			),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/admin/users/{id}", "AdminUsers", "获取用户认证详情",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeType[resp.UserAuthDetailResponse](),
		),
		op("PUT", "/api/v0/admin/users/{id}/login-credentials", "AdminUsers", "设置后台登录凭据",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withJSONBodyType[req.AdminLoginCredentialsRequest](),
			withEnvelopeResponse(messageSchema()),
			withErrors(400, 409),
		),
		op("POST", "/api/v0/admin/users/{id}/kick", "AdminUsers", "踢下线用户",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeResponse(messageWithCountSchema("deleted_session_count")),
		),
		op("POST", "/api/v0/admin/users/{id}/ban", "AdminUsers", "封禁用户",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withJSONBodyType[req.BanUserRequest](),
			withEnvelopeResponse(messageWithCountSchema("deleted_session_count")),
		),
		op("POST", "/api/v0/admin/users/{id}/unban", "AdminUsers", "解封用户",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/api/v0/admin/users/{id}/features", "AdminUsers", "获取用户功能明细",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeResponse(arraySchema(typeSchema[resp.UserFeatureInfo]())),
		),
		op("GET", "/api/v0/admin/rbac/roles", "RBAC", "获取角色列表",
			withSecurity(constant.PermissionUserManage),
			withEnvelopeResponse(arraySchema(typeSchema[resp.RoleWithUsersResponse]())),
		),
		op("GET", "/api/v0/admin/rbac/roles/permissions", "RBAC", "获取角色与权限映射",
			withSecurity(constant.PermissionUserManage),
			withEnvelopeType[resp.RolesWithPermissionsResponse](),
		),
		op("POST", "/api/v0/admin/rbac/roles", "RBAC", "创建角色",
			withSecurity(constant.PermissionUserManage),
			withJSONBodyType[req.CreateRoleRequest](),
			withEnvelopeType[models.Role](),
		),
		op("PUT", "/api/v0/admin/rbac/roles/{id}", "RBAC", "更新角色",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "角色 ID")),
			withJSONBodyType[req.UpdateRoleRequest](),
			withEnvelopeResponse(objSchema(field("id", int64Schema()))),
		),
		op("GET", "/api/v0/admin/rbac/permissions", "RBAC", "获取权限列表",
			withSecurity(constant.PermissionUserManage),
			withEnvelopeResponse(arraySchema(typeSchema[models.Permission]())),
		),
		op("POST", "/api/v0/admin/rbac/permissions", "RBAC", "创建权限",
			withSecurity(constant.PermissionUserManage),
			withJSONBodyType[req.CreatePermissionRequest](),
			withEnvelopeType[models.Permission](),
		),
		op("POST", "/api/v0/admin/rbac/roles/{id}/permissions", "RBAC", "重置角色权限",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "角色 ID")),
			withJSONBodyType[req.UpdateRolePermissionsRequest](),
			withEnvelopeResponse(objSchema(field("role_id", int64Schema()))),
		),
		op("POST", "/api/v0/admin/rbac/users/{id}/roles", "RBAC", "更新用户角色",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withJSONBodyType[req.UpdateUserRolesRequest](),
			withEnvelopeResponse(objSchema(field("user_id", int64Schema()))),
		),
		op("GET", "/api/v0/admin/rbac/users/{id}/permissions", "RBAC", "获取用户权限列表",
			withSecurity(constant.PermissionUserManage),
			withParams(pathIntParam("id", "用户 ID")),
			withEnvelopeResponse(objSchema(
				field("user_id", int64Schema()),
				field("permissions", arraySchema(stringSchema())),
			)),
		),
		op("DELETE", "/api/v0/admin/materials/{md5}", "Materials", "管理员删除资料",
			withSecurity(constant.PermissionMaterialManage),
			withParams(pathStringParam("md5", "资料 MD5")),
			withEnvelopeResponse(messageSchema()),
		),
		op("PUT", "/api/v0/admin/material-desc/{md5}", "Materials", "管理员更新资料描述",
			withSecurity(constant.PermissionMaterialManage),
			withParams(pathStringParam("md5", "资料 MD5")),
			withJSONBodyType[req.MaterialDescUpdateRequest](),
			withEnvelopeResponse(messageSchema()),
		),
		op("GET", "/{bucketName}/{proxyPath}", "Proxy", "MinIO 反向代理",
			withRouterMethod("ANY"),
			withDescription("运行时真实路由为 `/{cfg.BucketName}/*proxyPath`。当前未加认证，直接透传 MinIO 返回。"),
			withParams(
				pathStringParam("bucketName", "对象存储桶名称，需与运行时配置一致"),
				pathStringParam("proxyPath", "对象路径，可包含多级目录"),
			),
			withBinaryResponse("*/*"),
		),
	)

	return ops
}
