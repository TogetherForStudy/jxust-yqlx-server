package constant

const (
	CacheKeyConversationInfo     = "conversation:info:%d:%d"     // userID:conversationID basic metadata cache
	CacheKeyConversationMessages = "conversation:messages:%d:%d" // userID:conversationID
	CacheKeyAgentCheckpoint      = "agent:checkpoint:%s"         // checkpointID
)
