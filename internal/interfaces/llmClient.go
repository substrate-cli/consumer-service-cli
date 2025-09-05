package interfaces

type LLMClient interface {
	CallPrecheck(prompt string) (string, error)
	CallConstructBackendPrompt(prompt string) (string, error)
}
