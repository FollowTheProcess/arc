package diagnostic_test

import (
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/test"
)

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		name string                // Name of the test case
		want string                // Expected result
		diag diagnostic.Diagnostic // Diagnostic under test
	}{
		{
			name: "empty",
			diag: diagnostic.Diagnostic{},
			want: "",
		},
		{
			name: "severity only",
			diag: diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning},
			want: "",
		},
		{
			name: "valid but invalid severity",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityInvalid,
				Message:  "Uh oh!",
				Span: source.Span{
					File:        source.NewFile("test.http", []byte("hello\nthere\nworld")),
					StartOffset: 1,  // 'e' in "hello"
					EndOffset:   14, // 'r' in "world"
				},
			},
			want: "",
		},
		{
			name: "valid warning",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Message:  "Uh oh!",
				Span: source.Span{
					File:        source.NewFile("test.http", []byte("hello\nthere\nworld")),
					StartOffset: 1,  // 'e' in "hello"
					EndOffset:   14, // 'r' in "world"
				},
			},
			want: "[warning] test.http:1:2: Uh oh!",
		},
		{
			name: "valid error",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Message:  "This is broken",
				Span: source.Span{
					File:        source.NewFile("test.http", []byte("hello\nthere")),
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
