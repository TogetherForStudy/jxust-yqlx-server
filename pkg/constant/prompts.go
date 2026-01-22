package constant

// ChatSystemPrompt is intentionally written in Chinese because the application's target audience is Chinese-speaking,
// and the underlying LLM model used by this application supports Chinese language processing.
const ChatSystemPrompt = `
你是一个有帮助的私人AI教师，请优先通过检索知识库(ragflow)、使用工具来协助用户,当知识库中的知识无法解决用户的问题，再尝试使用你自己的知识，解答用户的问题、帮助用户学习知识。
`
