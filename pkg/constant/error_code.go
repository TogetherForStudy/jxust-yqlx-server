package constant

import "net/http"

type ResCode int

type ErrorMeta struct {
	HTTPStatus int
	Message    string
}

const SuccessCode ResCode = 0

const (
	CommonRouteNotFound      ResCode = 10001
	CommonMethodNotAllowed   ResCode = 10002
	CommonBadRequest         ResCode = 10003
	CommonNotFound           ResCode = 10004
	CommonConflict           ResCode = 10005
	CommonForbidden          ResCode = 10006
	CommonUnauthorized       ResCode = 10007
	CommonInternal           ResCode = 10008
	CommonServiceUnavailable ResCode = 10009
	CommonUserNotFound       ResCode = 10010
	CommonRequestPanicked    ResCode = 10011
)

const (
	AuthMissingUserContext          ResCode = 11001
	AuthInvalidAuthorizationHeader  ResCode = 11002
	AuthInvalidToken                ResCode = 11003
	AuthInvalidTokenType            ResCode = 11004
	AuthInvalidTokenClaims          ResCode = 11005
	AuthAccountBlocked              ResCode = 11006
	AuthSessionInvalid              ResCode = 11007
	AuthCacheUnavailable            ResCode = 11008
	AuthStateReadFailed             ResCode = 11009
	AuthStateParseFailed            ResCode = 11010
	AuthRefreshTokenInvalid         ResCode = 11011
	AuthRefreshTokenTypeInvalid     ResCode = 11012
	AuthRefreshTokenSessionNotFound ResCode = 11013
	AuthRefreshTokenExpired         ResCode = 11014
	AuthMissingSessionInfo          ResCode = 11015
	AuthUnsupportedTestUserType     ResCode = 11016
	AuthWechatLoginFailed           ResCode = 11017
	AuthAccountDisabled             ResCode = 11018
	AuthAccountTempBanned           ResCode = 11019
	AuthAccountKicked               ResCode = 11020
	AuthAdminLoginFailed            ResCode = 11021
	AuthAdminPasswordInvalid        ResCode = 11022
	AuthAdminTargetRoleInvalid      ResCode = 11023
	AuthAdminPhoneConflict          ResCode = 11024
	AuthAdminPhoneRequired          ResCode = 11025
)

// 12xxx: 会话相关
const (
	ConversationNotFound        ResCode = 12001
	ConversationMessageRequired ResCode = 12002
)

// 13xxx: 配置相关
const (
	ConfigKeyExists   ResCode = 13001
	ConfigKeyNotFound ResCode = 13002
)

// 20xxx: 投稿相关
const (
	ContributionNotFound            ResCode = 20001
	ContributionForbidden           ResCode = 20002
	ContributionReviewStatusInvalid ResCode = 20003
)

// 21xxx: 倒数日相关
const (
	CountdownTargetDateInvalid ResCode = 21001
	CountdownNotAccessible     ResCode = 21002
)

// 22xxx: 课程表相关
const (
	CourseTableClassNotSet              ResCode = 22001
	CourseTableScheduleNotFound         ResCode = 22002
	CourseTableClassNotFound            ResCode = 22003
	CourseTablePersonalScheduleNotFound ResCode = 22004
	CourseTableBindLimitReached         ResCode = 22005
)

// 23xxx: 功能白名单相关
const (
	FeatureNotFound         ResCode = 23001
	FeatureIdentifierExists ResCode = 23002
)

// 24xxx: 英雄榜相关
const (
	HeroNameExists ResCode = 24001
	HeroNotFound   ResCode = 24002
)

// 25xxx: 资料相关
const (
	MaterialNotFound            ResCode = 25001
	MaterialDescriptionNotFound ResCode = 25002
)

// 26xxx: 通知相关
const (
	NotificationNotFound            ResCode = 26001
	NotificationDeletedCannotModify ResCode = 26002
	NotificationNoPermissionModify  ResCode = 26003
	NotificationDraftOnlyPublish    ResCode = 26004
	NotificationNotPublished        ResCode = 26005
	NotificationReviewStatusInvalid ResCode = 26006
	NotificationAlreadyReviewed     ResCode = 26007
	NotificationOnlyPublishedCanPin ResCode = 26008
	NotificationAlreadyPinned       ResCode = 26009
	NotificationNotPinned           ResCode = 26010
	CategoryNotFound                ResCode = 26011
)

// 27xxx: 积分相关
const (
	PointsInsufficient ResCode = 27001
)

// 28xxx: 题库相关
const (
	QuestionNotFound ResCode = 28001
	QuestionDisabled ResCode = 28002
)

// 29xxx: 评价相关
const (
	ReviewDuplicate ResCode = 29001
	ReviewApproved  ResCode = 29002
	ReviewNotFound  ResCode = 29003
)

// 30xxx: 学习任务相关
const (
	StudyTaskDeadlineInvalid ResCode = 30001
	StudyTaskDateInvalid     ResCode = 30002
	StudyTaskNotAccessible   ResCode = 30003
)

// 31xxx: 对象存储相关
const (
	URIInvalid         ResCode = 31001
	TokenSecretMissing ResCode = 31002
)

// 32xxx: 词典相关
const (
	DictionaryRandomWordFailed ResCode = 32001
)

// 33xxx: 挂科率相关
const (
	FailRateQueryFailed ResCode = 33001
)

// 34xxx: 统计相关
const (
	StatServiceUnavailable ResCode = 34001
)

// 35xxx: 文件存储相关
const (
	StoreFileUploadFailed      ResCode = 35001
	StoreFileOpenFailed        ResCode = 35002
	StoreInvalidTags           ResCode = 35003
	StoreFileStoreFailed       ResCode = 35004
	StoreFileDeleteFailed      ResCode = 35005
	StoreFileListFailed        ResCode = 35006
	StoreExpiredFileListFailed ResCode = 35007
	StoreFileURLFailed         ResCode = 35008
	StoreFileNotFound          ResCode = 35009
	StoreFileStreamFailed      ResCode = 35010
)

// 36xxx: 用户活跃度相关
const (
	UserActivityQueryFailed ResCode = 36001
)

var ErrorMetaMap = map[ResCode]ErrorMeta{
	SuccessCode:                         {HTTPStatus: http.StatusOK, Message: "Success"},
	CommonRouteNotFound:                 {HTTPStatus: http.StatusNotFound, Message: "路由不存在"},
	CommonMethodNotAllowed:              {HTTPStatus: http.StatusMethodNotAllowed, Message: "请求方法不允许"},
	CommonBadRequest:                    {HTTPStatus: http.StatusBadRequest, Message: "请求参数错误"},
	CommonNotFound:                      {HTTPStatus: http.StatusNotFound, Message: "资源不存在"},
	CommonConflict:                      {HTTPStatus: http.StatusConflict, Message: "请求冲突"},
	CommonForbidden:                     {HTTPStatus: http.StatusForbidden, Message: "权限不足"},
	CommonUnauthorized:                  {HTTPStatus: http.StatusUnauthorized, Message: "未授权"},
	CommonInternal:                      {HTTPStatus: http.StatusInternalServerError, Message: "服务器内部错误"},
	CommonServiceUnavailable:            {HTTPStatus: http.StatusServiceUnavailable, Message: "服务暂不可用"},
	CommonUserNotFound:                  {HTTPStatus: http.StatusNotFound, Message: "用户不存在"},
	CommonRequestPanicked:               {HTTPStatus: http.StatusInternalServerError, Message: "服务器内部异常"},
	AuthMissingUserContext:              {HTTPStatus: http.StatusUnauthorized, Message: "未获取到用户信息"},
	AuthInvalidAuthorizationHeader:      {HTTPStatus: http.StatusUnauthorized, Message: "无效的 Authorization 头"},
	AuthInvalidToken:                    {HTTPStatus: http.StatusUnauthorized, Message: "无效的 Token"},
	AuthInvalidTokenType:                {HTTPStatus: http.StatusUnauthorized, Message: "Token 类型无效"},
	AuthInvalidTokenClaims:              {HTTPStatus: http.StatusUnauthorized, Message: "无效的 Token Claims"},
	AuthAccountBlocked:                  {HTTPStatus: http.StatusUnauthorized, Message: "用户账号已被封禁"},
	AuthSessionInvalid:                  {HTTPStatus: http.StatusUnauthorized, Message: "当前会话已失效"},
	AuthCacheUnavailable:                {HTTPStatus: http.StatusServiceUnavailable, Message: "鉴权缓存未初始化"},
	AuthStateReadFailed:                 {HTTPStatus: http.StatusServiceUnavailable, Message: "鉴权状态读取失败"},
	AuthStateParseFailed:                {HTTPStatus: http.StatusServiceUnavailable, Message: "鉴权状态解析失败"},
	AuthRefreshTokenInvalid:             {HTTPStatus: http.StatusUnauthorized, Message: "无效的 RefreshToken"},
	AuthRefreshTokenTypeInvalid:         {HTTPStatus: http.StatusUnauthorized, Message: "RefreshToken 类型无效"},
	AuthRefreshTokenSessionNotFound:     {HTTPStatus: http.StatusUnauthorized, Message: "RefreshToken 会话不存在"},
	AuthRefreshTokenExpired:             {HTTPStatus: http.StatusUnauthorized, Message: "RefreshToken 已失效"},
	AuthMissingSessionInfo:              {HTTPStatus: http.StatusUnauthorized, Message: "缺少会话信息"},
	AuthUnsupportedTestUserType:         {HTTPStatus: http.StatusBadRequest, Message: "不支持的测试用户类型"},
	AuthWechatLoginFailed:               {HTTPStatus: http.StatusBadGateway, Message: "微信登录失败"},
	AuthAccountDisabled:                 {HTTPStatus: http.StatusUnauthorized, Message: "用户账号已被禁用"},
	AuthAccountTempBanned:               {HTTPStatus: http.StatusUnauthorized, Message: "用户账号已被临时封禁"},
	AuthAccountKicked:                   {HTTPStatus: http.StatusUnauthorized, Message: "账号已被下线，请稍后重试"},
	AuthAdminLoginFailed:                {HTTPStatus: http.StatusUnauthorized, Message: "手机号或密码错误"},
	AuthAdminPasswordInvalid:            {HTTPStatus: http.StatusBadRequest, Message: "后台密码必须至少8位且包含字母和数字"},
	AuthAdminTargetRoleInvalid:          {HTTPStatus: http.StatusBadRequest, Message: "目标用户不是后台账号"},
	AuthAdminPhoneConflict:              {HTTPStatus: http.StatusConflict, Message: "后台登录手机号已被占用"},
	AuthAdminPhoneRequired:              {HTTPStatus: http.StatusBadRequest, Message: "后台登录手机号不能为空"},
	ConversationNotFound:                {HTTPStatus: http.StatusNotFound, Message: "会话不存在"},
	ConversationMessageRequired:         {HTTPStatus: http.StatusBadRequest, Message: "新会话必须提供消息内容"},
	ConfigKeyExists:                     {HTTPStatus: http.StatusConflict, Message: "配置键已存在"},
	ConfigKeyNotFound:                   {HTTPStatus: http.StatusNotFound, Message: "配置项不存在"},
	ContributionNotFound:                {HTTPStatus: http.StatusNotFound, Message: "投稿不存在"},
	ContributionForbidden:               {HTTPStatus: http.StatusForbidden, Message: "无权限"},
	ContributionReviewStatusInvalid:     {HTTPStatus: http.StatusConflict, Message: "只能审核待审核状态的投稿"},
	CountdownTargetDateInvalid:          {HTTPStatus: http.StatusBadRequest, Message: "目标日期格式错误"},
	CountdownNotAccessible:              {HTTPStatus: http.StatusNotFound, Message: "倒数日不存在或无权限访问"},
	CourseTableClassNotSet:              {HTTPStatus: http.StatusConflict, Message: "用户尚未设置班级信息"},
	CourseTableScheduleNotFound:         {HTTPStatus: http.StatusNotFound, Message: "未找到该班级在指定学期的课程表"},
	CourseTableClassNotFound:            {HTTPStatus: http.StatusNotFound, Message: "指定的班级不存在"},
	CourseTablePersonalScheduleNotFound: {HTTPStatus: http.StatusNotFound, Message: "未找到个人课表数据"},
	CourseTableBindLimitReached:         {HTTPStatus: http.StatusConflict, Message: "仅可绑定2次"},
	FeatureNotFound:                     {HTTPStatus: http.StatusNotFound, Message: "功能不存在"},
	FeatureIdentifierExists:             {HTTPStatus: http.StatusConflict, Message: "功能标识已存在"},
	HeroNameExists:                      {HTTPStatus: http.StatusConflict, Message: "名称已存在"},
	HeroNotFound:                        {HTTPStatus: http.StatusNotFound, Message: "未找到"},
	MaterialNotFound:                    {HTTPStatus: http.StatusNotFound, Message: "资料不存在"},
	MaterialDescriptionNotFound:         {HTTPStatus: http.StatusNotFound, Message: "资料描述不存在"},
	NotificationNotFound:                {HTTPStatus: http.StatusNotFound, Message: "通知不存在"},
	NotificationDeletedCannotModify:     {HTTPStatus: http.StatusConflict, Message: "已删除的通知不能修改"},
	NotificationNoPermissionModify:      {HTTPStatus: http.StatusForbidden, Message: "无权限修改"},
	NotificationDraftOnlyPublish:        {HTTPStatus: http.StatusConflict, Message: "只能发布草稿状态的通知"},
	NotificationNotPublished:            {HTTPStatus: http.StatusConflict, Message: "通知未发布"},
	NotificationReviewStatusInvalid:     {HTTPStatus: http.StatusConflict, Message: "只能审核待审核状态的通知"},
	NotificationAlreadyReviewed:         {HTTPStatus: http.StatusConflict, Message: "您已经审核过该通知"},
	NotificationOnlyPublishedCanPin:     {HTTPStatus: http.StatusConflict, Message: "只有已发布的通知才能置顶"},
	NotificationAlreadyPinned:           {HTTPStatus: http.StatusConflict, Message: "通知已经置顶"},
	NotificationNotPinned:               {HTTPStatus: http.StatusConflict, Message: "通知未置顶"},
	CategoryNotFound:                    {HTTPStatus: http.StatusNotFound, Message: "分类不存在"},
	PointsInsufficient:                  {HTTPStatus: http.StatusConflict, Message: "积分不足"},
	QuestionNotFound:                    {HTTPStatus: http.StatusNotFound, Message: "题目不存在"},
	QuestionDisabled:                    {HTTPStatus: http.StatusConflict, Message: "题目已禁用"},
	ReviewDuplicate:                     {HTTPStatus: http.StatusConflict, Message: "您已经评价过该教师的这门课程"},
	ReviewApproved:                      {HTTPStatus: http.StatusConflict, Message: "评价已审核通过，无需重复审核"},
	ReviewNotFound:                      {HTTPStatus: http.StatusNotFound, Message: "评价不存在"},
	StudyTaskDeadlineInvalid:            {HTTPStatus: http.StatusBadRequest, Message: "截止时间格式错误"},
	StudyTaskDateInvalid:                {HTTPStatus: http.StatusBadRequest, Message: "截止日期格式错误"},
	StudyTaskNotAccessible:              {HTTPStatus: http.StatusNotFound, Message: "学习任务不存在或无权限访问"},
	URIInvalid:                          {HTTPStatus: http.StatusBadRequest, Message: "uri 必须以 / 开头，并且不可为空"},
	TokenSecretMissing:                  {HTTPStatus: http.StatusInternalServerError, Message: "对象存储配置缺失"},
	DictionaryRandomWordFailed:          {HTTPStatus: http.StatusInternalServerError, Message: "获取随机单词失败"},
	FailRateQueryFailed:                 {HTTPStatus: http.StatusInternalServerError, Message: "查询挂科率失败"},
	StatServiceUnavailable:              {HTTPStatus: http.StatusServiceUnavailable, Message: "统计服务暂不可用"},
	StoreFileUploadFailed:               {HTTPStatus: http.StatusBadRequest, Message: "上传文件失败"},
	StoreFileOpenFailed:                 {HTTPStatus: http.StatusInternalServerError, Message: "打开上传文件失败"},
	StoreInvalidTags:                    {HTTPStatus: http.StatusBadRequest, Message: "标签格式错误"},
	StoreFileStoreFailed:                {HTTPStatus: http.StatusInternalServerError, Message: "存储文件失败"},
	StoreFileDeleteFailed:               {HTTPStatus: http.StatusInternalServerError, Message: "删除文件失败"},
	StoreFileListFailed:                 {HTTPStatus: http.StatusInternalServerError, Message: "获取文件列表失败"},
	StoreExpiredFileListFailed:          {HTTPStatus: http.StatusInternalServerError, Message: "获取过期文件列表失败"},
	StoreFileURLFailed:                  {HTTPStatus: http.StatusInternalServerError, Message: "生成文件链接失败"},
	StoreFileNotFound:                   {HTTPStatus: http.StatusNotFound, Message: "文件不存在"},
	StoreFileStreamFailed:               {HTTPStatus: http.StatusInternalServerError, Message: "文件流传输失败"},
	UserActivityQueryFailed:             {HTTPStatus: http.StatusInternalServerError, Message: "查询登录天数失败"},
}

func LookupErrorMeta(code ResCode) (ErrorMeta, bool) {
	meta, ok := ErrorMetaMap[code]
	return meta, ok
}

func DefaultHTTPStatus(code ResCode) int {
	if meta, ok := LookupErrorMeta(code); ok {
		return meta.HTTPStatus
	}
	return http.StatusInternalServerError
}

func DefaultErrorMessage(code ResCode) string {
	if meta, ok := LookupErrorMeta(code); ok {
		return meta.Message
	}
	return ErrorMetaMap[CommonInternal].Message
}
