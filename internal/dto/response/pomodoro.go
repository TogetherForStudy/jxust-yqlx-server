package response

// PomodoroRankingItem 番茄钟排名项
type PomodoroRankingItem struct {
	Rank          int    `json:"rank"`
	Nickname      string `json:"nickname"`
	PomodoroCount uint   `json:"pomodoro_count"`
}
