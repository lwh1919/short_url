package sign

type SignHandler interface {
	GenerateSign(data map[string]any, apiKey string) (string, error)
}
