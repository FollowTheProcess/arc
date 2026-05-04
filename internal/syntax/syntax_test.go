package syntax_test

import (
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax"
	"go.followtheprocess.codes/test"
)

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		name string            // Name of the test case
		want string            // Expected result
		diag syntax.Diagnostic // Diagnostic under test
	}{
		{
			name: "empty",
			diag: syntax.Diagnostic{},
			want: "",
		},
		{
			name: "severity only",
			diag: syntax.Diagnostic{Severity: syntax.SeverityWarning},
			want: "",
		},
		{
			name: "valid but invalid severity",
			diag: syntax.Diagnostic{
				Severity: syntax.SeverityInvalid,
				Message:  "Uh oh!",
				Span: syntax.Span{
					File:        syntax.NewSourceFile("test.http", []byte("hello\nthere\nworld")),
					StartOffset: 1,  // 'e' in "hello"
					EndOffset:   14, // 'r' in "world"
				},
			},
			want: "",
		},
		{
			name: "valid warning",
			diag: syntax.Diagnostic{
				Severity: syntax.SeverityWarning,
				Message:  "Uh oh!",
				Span: syntax.Span{
					File:        syntax.NewSourceFile("test.http", []byte("hello\nthere\nworld")),
					StartOffset: 1,  // 'e' in "hello"
					EndOffset:   14, // 'r' in "world"
				},
			},
			want: "[warning] test.http:1:2: Uh oh!",
		},
		{
			name: "valid error",
			diag: syntax.Diagnostic{
				Severity: syntax.SeverityError,
				Message:  "This is broken",
				Span: syntax.Span{
					File:        syntax.NewSourceFile("test.http", []byte("hello\nthere")),
					StartOffset: 6, // 't' in "there"
					EndOffset:   6,
				},
			},
			want: "[error] test.http:2:1: This is broken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.diag.String()
			test.Equal(t, got, tt.want)
		})
	}
}
