package diagnostic_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"os"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/diagnostic"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
)

var (
	update = flag.Bool("update", false, "Update snapshots")
	clean  = flag.Bool("clean", false, "Re-generate all snapshots from scratch")
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

func TestDiagnosticJSON(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	tests := []struct {
		name string                // Name of the test case
		diag diagnostic.Diagnostic // Diagnostic under test
	}{
		{
			name: "empty",
			diag: diagnostic.Diagnostic{},
		},
		{
			name: "severity only",
			diag: diagnostic.Diagnostic{Severity: diagnostic.SeverityWarning},
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot.New(t, snapshot.Update(*update), snapshot.Clean(*clean))

			content, err := json.Marshal(tt.diag, json.Deterministic(true), jsontext.WithIndent("  "))
			test.Ok(t, err)

			snap.Snap(string(content))
		})
	}
}
