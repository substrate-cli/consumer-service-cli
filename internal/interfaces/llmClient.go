package interfaces

type LLMClient interface {
	CallPrecheck(prompt string) (string, error)
	CallConstructBackendPrompt(prompt string) (string, error)
	VisionAnalysis(screenshot string, url string, title string, isRepo bool) (string, error)
	CallGithubTreeScan(prompt string) (string, error)
	CallPrePromptForGithubClone(description string) (string, error)
}
