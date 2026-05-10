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
	invalidFile := source.NewFile("test.http", []byte("POST /api\n"))
	warningFile := source.NewFile("test.http", []byte("GET http://api.local/users\n"))
	errorFile := source.NewFile("test.http", []byte("GETT /users HTTP/1.1\n"))

	// Extra stuff like Labels and Fixes aren't shown in .String() output
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
				Message:  "POST without Content-Type",
				Span: source.Span{
					File:        invalidFile,
					StartOffset: 0, // 'P' in "POST"
					EndOffset:   4, // end of "POST"
				},
			},
			want: "",
		},
		{
			name: "valid warning",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Message:  "use https for transport security",
				Span: source.Span{
					File:        warningFile,
					StartOffset: 4, // 'h' in "http"
					EndOffset:   8, // end of "http"
				},
				Labels: []diagnostic.Label{
					{
						Message: "this URL will be sent in plaintext",
						Span: source.Span{
							File:        warningFile,
							StartOffset: 4,  // 'h' in "http"
							EndOffset:   26, // end of "/users"
						},
					},
				},
				Fixes: []diagnostic.Fix{
					{
						Message: "replace with 'https'",
						Edits: []diagnostic.Edit{
							{
								Replacement: "https",
								Span: source.Span{
									File:        warningFile,
									StartOffset: 4,
									EndOffset:   8,
								},
							},
						},
					},
				},
			},
			want: "[warning] test.http:1:5-9: use https for transport security",
		},
		{
			name: "valid error",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Message:  "unknown HTTP method 'GETT'",
				Span: source.Span{
					File:        errorFile,
					StartOffset: 0, // 'G' in "GETT"
					EndOffset:   4, // end of "GETT"
				},
				Labels: []diagnostic.Label{
					{
						Message: "applies to this request",
						Span: source.Span{
							File:        errorFile,
							StartOffset: 5,  // '/' in "/users"
							EndOffset:   11, // end of "/users"
						},
					},
				},
				Fixes: []diagnostic.Fix{
					{
						Message: "did you mean 'GET'?",
						Edits: []diagnostic.Edit{
							{
								Replacement: "GET",
								Span: source.Span{
									File:        errorFile,
									StartOffset: 0,
									EndOffset:   4,
								},
							},
						},
					},
				},
			},
			want: "[error] test.http:1:1-5: unknown HTTP method 'GETT'",
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

	invalidFile := source.NewFile("test.http", []byte("DELETE /\n"))
	warningFile := source.NewFile(
		"test.http",
		[]byte("GET /\nAccept: application/json\nAccept: text/plain\n"),
	)
	errorFile := source.NewFile("test.http", []byte("GET {{url}}/users\nHost: {{url}}\n"))

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
			diag: diagnostic.Diagnostic{Severity: diagnostic.SeverityError},
		},
		{
			name: "valid but invalid severity",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityInvalid,
				Message:  "DELETE with empty body",
				Span: source.Span{
					File:        invalidFile,
					StartOffset: 0, // 'D' in "DELETE"
					EndOffset:   6, // end of "DELETE"
				},
			},
		},
		{
			name: "valid warning",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Message:  "duplicate 'Accept' header",
				Span: source.Span{
					File:        warningFile,
					StartOffset: 31, // 'A' in second "Accept"
					EndOffset:   37, // end of second "Accept"
				},
				Labels: []diagnostic.Label{
					{
						Message: "first declared here",
						Span: source.Span{
							File:        warningFile,
							StartOffset: 6,  // 'A' in first "Accept"
							EndOffset:   12, // end of first "Accept"
						},
					},
				},
				Fixes: []diagnostic.Fix{
					{
						Message: "remove the duplicate header",
						Edits: []diagnostic.Edit{
							{
								// Replacing the entire line including its trailing
								// newline removes the header without leaving a blank line.
								Replacement: "",
								Span: source.Span{
									File:        warningFile,
									StartOffset: 31,
									EndOffset:   50,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "valid error",
			diag: diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Message:  "undefined variable 'url'",
				Span: source.Span{
					File:        errorFile,
					StartOffset: 4, // first '{{url}}'
					EndOffset:   11,
				},
				Labels: []diagnostic.Label{
					{
						Message: "also referenced here",
						Span: source.Span{
							File:        errorFile,
							StartOffset: 24, // second '{{url}}'
							EndOffset:   31,
						},
					},
				},
				Fixes: []diagnostic.Fix{
					{
						Message: "rename 'url' to 'baseUrl'",
						Edits: []diagnostic.Edit{
							{
								Replacement: "baseUrl",
								Span: source.Span{
									File:        errorFile,
									StartOffset: 6, // 'url' inside first '{{...}}'
									EndOffset:   9,
								},
							},
							{
								Replacement: "baseUrl",
								Span: source.Span{
									File:        errorFile,
									StartOffset: 26, // 'url' inside second '{{...}}'
									EndOffset:   29,
								},
							},
						},
					},
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
