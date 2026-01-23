package constant

import "time"

const (
	CacheKeyConversationInfo     = "conversation:info:%d:%d"     // userID:conversationID basic metadata cache
	CacheKeyConversationMessages = "conversation:messages:%d:%d" // userID:conversationID
	CacheKeyAgentCheckpoint      = "agent:checkpoint:%s"         // checkpointID
	CacheKeyUserFeatures         = "user_features:%d"            // 用户功能列表缓存
	CacheKeyFeatureEnabled       = "feature_enabled:%s"          // 功能全局开关缓存
)

// Cache TTL
const (
	UserFeaturesCacheTTL   = 5 * time.Minute  // 用户功能列表缓存5分钟
	FeatureEnabledCacheTTL = 10 * time.Minute // 功能开关缓存10分钟
)
