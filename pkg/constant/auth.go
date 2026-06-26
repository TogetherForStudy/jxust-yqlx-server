package constant

import "time"

const (
	AuthTokenTypeAccess  = "access"
	AuthTokenTypeRefresh = "refresh"

	AuthBlockTypeKick      = "kick"
	AuthBlockTypeTempBan   = "temp_ban"
	AuthBlockTypePermanent = "permanent_ban"

	AuthClientTypeMiniProgram = "wechat_mini_program"
	AuthClientTypeUnknown     = "unknown"

	AuthDeviceTypeIOS     = "ios"
	AuthDeviceTypeAndroid = "android"
	AuthDeviceTypeWindows = "windows"
	AuthDeviceTypeMac     = "mac"
	AuthDeviceTypeLinux   = "linux"
	AuthDeviceTypeUnknown = "unknown"

	AuthContextSessionID = "auth_session_id"
	AuthContextTokenJTI  = "auth_token_jti"
	AuthContextTokenIAT  = "auth_token_iat"
)

const (
	AuthSessionKeyFormat        = "auth:session:%s"
	AuthUserSessionsKeyFormat   = "auth:user_sessions:%d"
	AuthRevokedSessionKeyFormat = "auth:revoked_session:%s"
	AuthRevokedBeforeKeyFormat  = "auth:revoked_before:%d"
	AuthBlockedKeyFormat        = "auth:blocked:%d"
)

const (
	DefaultAccessTokenTTL  = 2 * time.Hour
	DefaultRefreshTokenTTL = 30 * 24 * time.Hour
)
