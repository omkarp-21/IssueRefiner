package providers

type Input struct {
	SystemPrompt string
	Owner        string
	Repo         string
	IssueNumber  int
	Title        string
	Body         string
}

type Result struct {
	Text         string
	InputTokens  int
	OutputTokens int
}
