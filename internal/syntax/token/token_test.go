package token_test

import (
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/token"
	"go.followtheprocess.codes/test"
)

func TestTokenString(t *testing.T) {
	tests := []struct {
		name string      // Name of the test case
		want string      // Expected String() value
		tok  token.Token // Token under test
	}{
		{
			name: "empty",
			tok:  token.Token{},
			want: "<Token::EOF start=0, end=0>",
		},
		{
			name: "error",
			tok:  token.Token{Kind: token.Error},
			want: "<Token::Error start=0, end=0>",
		},
		{
			name: "text",
			tok:  token.Token{Kind: token.Text, Start: 1, End: 12},
			want: "<Token::Text start=1, end=12>",
		},
		{
			name: "lparen",
			tok:  token.Token{Kind: token.LParen, Start: 127, End: 128},
			want: "<Token::LParen start=127, end=128>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tok.String()
			test.Equal(t, got, tt.want)
		})
	}
}
