package report

import (
	"github.com/tiktoken-go/tokenizer"
)

// TokenCounter wraps tiktoken for token counting
type TokenCounter struct {
	enc tokenizer.Codec
}

// NewTokenCounter creates a new token counter using cl100k_base encoding
func NewTokenCounter() (*TokenCounter, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return nil, err
	}
	return &TokenCounter{enc: enc}, nil
}

// Count returns the number of tokens in the given text
func (tc *TokenCounter) Count(text string) int {
	tokens, _, _ := tc.enc.Encode(text)
	return len(tokens)
}

