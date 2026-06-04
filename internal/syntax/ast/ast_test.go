package ast_test

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.followtheprocess.codes/arc/internal/syntax/ast"
	"go.followtheprocess.codes/arc/internal/syntax/source"
	"go.followtheprocess.codes/snapshot"
	"go.followtheprocess.codes/test"
)

var (
	update = flag.Bool("update", false, "Update golden files and snapshots")
	clean  = flag.Bool("clean", false, "Regenerate snapshots from scratch")
)

func TestDump(t *testing.T) {
	// Force colour for diffs but only locally
	test.ColorEnabled(os.Getenv("CI") == "")

	tests := []struct {
		node ast.Node // Node under test
		name string   // Name of the test case
	}{
		{
			name: "nil",
			node: nil,
		},
		{
			// Dump recurses, so dumping the fixture exercises every node type.
			name: "file",
			node: tree(t),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot.New(
				t,
				snapshot.Clean(*clean),
				snapshot.Update(*update),
				snapshot.WithFormatter(snapshot.TextFormatter()),
			)

			snap.Snap(ast.Dump(tt.node))
		})
	}
}

func TestInspectVisitsEveryNodeDepthFirst(t *testing.T) {
	var got []string

	ast.Inspect(tree(t), func(node ast.Node) bool {
		if node != nil {
			got = append(got, fmt.Sprintf("%T", node))
		}

		return true
	})

	want := strings.Join([]string{
		"ast.File",
		"ast.Comment",
		// @timeout = 30s
		"ast.Directive",
		"ast.Ident",
		"ast.TextLiteral",
		// @base = {{ $env.BASE_URL }}
		"ast.Directive",
		"ast.Ident",
		"ast.Interp",
		"ast.Selector",
		"ast.Builtin",
		"ast.Ident",
		"ast.Ident",
		"ast.Request",
		"*ast.Comment",
		"*ast.Ident",
		"ast.Ident",
		// URL: https://example.com/users/{{ $random.int(1, 100) }}
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Call",
		"ast.Selector",
		"ast.Builtin",
		"ast.Ident",
		"ast.Ident",
		"ast.NumberLiteral",
		"ast.NumberLiteral",
		"*ast.HTTPVersion",
		"ast.NumberLiteral",
		// Authorization: Bearer {{ token }}
		"ast.Header",
		"ast.Ident",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		// X-Request-Id: {{ $uuid }}
		"ast.Header",
		"ast.Ident",
		"ast.Interp",
		"ast.Builtin",
		"ast.Ident",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: application/json
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		// {"name": "{{ userName }}"}
		"ast.BodyInline",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// <@ ./payload.json
		"ast.BodyFile",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: multipart/form-data; boundary=xyz
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.BodyMultipart",
		"ast.MultipartPart",
		// Content-Disposition: form-data; name="file"; filename="input.txt"
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		// < ./input.txt
		"ast.BodyFile",
		"ast.Template",
		"ast.TextLiteral",
		// hello world
		"ast.MultipartPart",
		"ast.BodyInline",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: application/x-www-form-urlencoded
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.BodyForm",
		// search={{ query }}
		"ast.FormField",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		// sort=desc
		"ast.FormField",
		"ast.TextLiteral",
		"ast.TextLiteral",
	}, "\n")
	test.Diff(t, strings.Join(got, "\n"), want)
}

func TestInspectStopsDescendingWhenCallbackReturnsFalse(t *testing.T) {
	var got []string

	ast.Inspect(tree(t), func(node ast.Node) bool {
		if node == nil {
			return true
		}

		got = append(got, fmt.Sprintf("%T", node))

		// Don't descend into the directive's children.
		_, isDirective := node.(ast.Directive)

		return !isDirective
	})

	want := strings.Join([]string{
		"ast.File",
		"ast.Comment",
		// Both directives are visited but not descended into.
		"ast.Directive",
		"ast.Directive",
		"ast.Request",
		"*ast.Comment",
		"*ast.Ident",
		"ast.Ident",
		// URL: https://example.com/users/{{ $random.int(1, 100) }}
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Call",
		"ast.Selector",
		"ast.Builtin",
		"ast.Ident",
		"ast.Ident",
		"ast.NumberLiteral",
		"ast.NumberLiteral",
		"*ast.HTTPVersion",
		"ast.NumberLiteral",
		// Authorization: Bearer {{ token }}
		"ast.Header",
		"ast.Ident",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		// X-Request-Id: {{ $uuid }}
		"ast.Header",
		"ast.Ident",
		"ast.Interp",
		"ast.Builtin",
		"ast.Ident",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: application/json
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		// {"name": "{{ userName }}"}
		"ast.BodyInline",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// <@ ./payload.json
		"ast.BodyFile",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: multipart/form-data; boundary=xyz
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.BodyMultipart",
		"ast.MultipartPart",
		// Content-Disposition: form-data; name="file"; filename="input.txt"
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		// < ./input.txt
		"ast.BodyFile",
		"ast.Template",
		"ast.TextLiteral",
		// hello world
		"ast.MultipartPart",
		"ast.BodyInline",
		"ast.Template",
		"ast.TextLiteral",
		"ast.Request",
		"*ast.Ident",
		"ast.Ident",
		"ast.TextLiteral",
		// Content-Type: application/x-www-form-urlencoded
		"ast.Header",
		"ast.Ident",
		"ast.TextLiteral",
		"ast.BodyForm",
		// search={{ query }}
		"ast.FormField",
		"ast.TextLiteral",
		"ast.Interp",
		"ast.Ident",
		// sort=desc
		"ast.FormField",
		"ast.TextLiteral",
		"ast.TextLiteral",
	}, "\n")
	test.Diff(t, strings.Join(got, "\n"), want)
}

// tree builds the AST for testdata/fixture.http, exercising every node type in
// the package. Node spans are derived from the fixture text by substring lookup
// so the fixture reads like real source rather than a table of byte offsets.
func tree(t *testing.T) ast.File {
	t.Helper()

	src, err := os.ReadFile(filepath.Join("testdata", "fixture.http"))
	test.Ok(t, err)

	file := source.NewFile("fixture.http", src)

	// span returns the span of the first occurrence of sub in the fixture.
	span := func(sub string) source.Span {
		i := bytes.Index(src, []byte(sub))
		test.True(t, i >= 0, test.Context("substring %q not found in fixture", sub))

		return source.Span{File: file, StartOffset: i, EndOffset: i + len(sub)}
	}

	// spanAfter returns the span of the first occurrence of sub at or after
	// the first occurrence of marker, for substrings that appear more than
	// once in the fixture.
	spanAfter := func(marker, sub string) source.Span {
		m := bytes.Index(src, []byte(marker))
		test.True(t, m >= 0, test.Context("marker %q not found in fixture", marker))

		i := bytes.Index(src[m:], []byte(sub))
		test.True(t, i >= 0, test.Context("substring %q not found after %q in fixture", sub, marker))

		start := m + i

		return source.Span{File: file, StartOffset: start, EndOffset: start + len(sub)}
	}

	// enclosing spans from the start of from to the end of to.
	enclosing := func(from, to source.Span) source.Span {
		return source.Span{File: file, StartOffset: from.StartOffset, EndOffset: to.EndOffset}
	}

	return ast.File{
		Statements: []ast.Statement{
			ast.Comment{Range: span("# config")},
			ast.Directive{
				Ident: ast.Ident{Range: span("timeout")},
				Value: ast.TextLiteral{Value: "30s", Range: span("30s")},
				Range: span("@timeout = 30s"),
			},
			ast.Directive{
				Ident: ast.Ident{Range: span("base")},
				// @base = {{ $env.BASE_URL }} -> a builtin selector.
				Value: ast.Interp{
					Inner: ast.Selector{
						Expr: ast.Builtin{
							Name:  ast.Ident{Range: span("env")},
							Range: span("$env"),
						},
						Sel:   ast.Ident{Range: span("BASE_URL")},
						Range: span("$env.BASE_URL"),
					},
					Range: span("{{ $env.BASE_URL }}"),
				},
				Range: span("@base = {{ $env.BASE_URL }}"),
			},
			ast.Request{
				Doc:    &ast.Comment{Range: span("# Fetch a user")},
				Name:   &ast.Ident{Range: span("get-user")},
				Method: ast.Ident{Range: span("GET")},
				// URL interleaves literal text with a builtin call.
				URL: ast.Template{
					Parts: []ast.Expression{
						ast.TextLiteral{
							Value: "https://example.com/users/",
							Range: span("https://example.com/users/"),
						},
						ast.Interp{
							Inner: ast.Call{
								Fun: ast.Selector{
									Expr: ast.Builtin{
										Name:  ast.Ident{Range: span("random")},
										Range: span("$random"),
									},
									Sel:   ast.Ident{Range: span("int")},
									Range: span("$random.int"),
								},
								Args: []ast.Expression{
									ast.NumberLiteral{Range: span("1")},
									ast.NumberLiteral{Range: span("100")},
								},
								Range: span("$random.int(1, 100)"),
							},
							Range: span("{{ $random.int(1, 100) }}"),
						},
					},
					Range: span("https://example.com/users/{{ $random.int(1, 100) }}"),
				},
				HTTPVersion: &ast.HTTPVersion{
					Version: ast.NumberLiteral{Range: span("1.1")},
					Range:   span("HTTP/1.1"),
				},
				Headers: []ast.Header{
					{
						Name: ast.Ident{Range: span("Authorization")},
						Value: ast.Template{
							Parts: []ast.Expression{
								ast.TextLiteral{Value: "Bearer ", Range: span("Bearer ")},
								ast.Interp{
									Inner: ast.Ident{Range: span("token")},
									Range: span("{{ token }}"),
								},
							},
							Range: span("Bearer {{ token }}"),
						},
						Range: span("Authorization: Bearer {{ token }}"),
					},
					{
						Name: ast.Ident{Range: span("X-Request-Id")},
						// A bare builtin as the whole header value.
						Value: ast.Interp{
							Inner: ast.Builtin{
								Name:  ast.Ident{Range: span("uuid")},
								Range: span("$uuid"),
							},
							Range: span("{{ $uuid }}"),
						},
						Range: span("X-Request-Id: {{ $uuid }}"),
					},
				},
				Range: enclosing(span("### get-user"), span("{{ $uuid }}")),
			},
			ast.Request{
				Name:   &ast.Ident{Range: span("create-user")},
				Method: ast.Ident{Range: span("POST")},
				URL: ast.TextLiteral{
					Value: "https://example.com/items",
					Range: span("https://example.com/items"),
				},
				Headers: []ast.Header{
					{
						Name: ast.Ident{Range: span("Content-Type")},
						Value: ast.TextLiteral{
							Value: "application/json",
							Range: span("application/json"),
						},
						Range: span("Content-Type: application/json"),
					},
				},
				// An inline body interleaving literal JSON with an interp.
				Body: ast.BodyInline{
					Content: ast.Template{
						Parts: []ast.Expression{
							ast.TextLiteral{Value: `{"name": "`, Range: span(`{"name": "`)},
							ast.Interp{
								Inner: ast.Ident{Range: span("userName")},
								Range: span("{{ userName }}"),
							},
							ast.TextLiteral{Value: `"}`, Range: span(`"}`)},
						},
						Range: span(`{"name": "{{ userName }}"}`),
					},
					Range: span(`{"name": "{{ userName }}"}`),
				},
				Range: enclosing(span("### create-user"), span(`{"name": "{{ userName }}"}`)),
			},
			ast.Request{
				Name:   &ast.Ident{Range: span("import-data")},
				Method: ast.Ident{Range: span("PUT")},
				URL: ast.TextLiteral{
					Value: "https://example.com/import",
					Range: span("https://example.com/import"),
				},
				// A templated file body with an encoding: <@latin1 ./payload.json.
				Body: ast.BodyFile{
					Path: ast.Template{
						Parts: []ast.Expression{
							ast.TextLiteral{Value: "./payload.json", Range: span("./payload.json")},
						},
						Range: span("./payload.json"),
					},
					Templated: true,
					Encoding:  "latin1",
					Range:     span("<@latin1 ./payload.json"),
				},
				Range: enclosing(span("### import-data"), span("<@latin1 ./payload.json")),
			},
			ast.Request{
				Name:   &ast.Ident{Range: span("upload")},
				Method: ast.Ident{Range: spanAfter("### upload", "POST")},
				URL: ast.TextLiteral{
					Value: "https://example.com/upload",
					Range: span("https://example.com/upload"),
				},
				Headers: []ast.Header{
					{
						Name: ast.Ident{Range: spanAfter("### upload", "Content-Type")},
						Value: ast.TextLiteral{
							Value: "multipart/form-data; boundary=xyz",
							Range: span("multipart/form-data; boundary=xyz"),
						},
						Range: span("Content-Type: multipart/form-data; boundary=xyz"),
					},
				},
				Body: ast.BodyMultipart{
					Boundary: "xyz",
					Parts: []ast.MultipartPart{
						{
							Headers: []ast.Header{
								{
									Name: ast.Ident{Range: span("Content-Disposition")},
									Value: ast.TextLiteral{
										Value: `form-data; name="file"; filename="input.txt"`,
										Range: span(`form-data; name="file"; filename="input.txt"`),
									},
									Range: span(`Content-Disposition: form-data; name="file"; filename="input.txt"`),
								},
							},
							// A part body can be any body kind, here a literal file ref.
							Body: ast.BodyFile{
								Path: ast.Template{
									Parts: []ast.Expression{
										ast.TextLiteral{Value: "./input.txt", Range: span("./input.txt")},
									},
									Range: span("./input.txt"),
								},
								Range: span("< ./input.txt"),
							},
							Range: enclosing(
								span(`Content-Disposition: form-data; name="file"; filename="input.txt"`),
								span("< ./input.txt"),
							),
						},
						{
							// A bare inline part with no headers of its own.
							Body: ast.BodyInline{
								Content: ast.Template{
									Parts: []ast.Expression{
										ast.TextLiteral{Value: "hello world", Range: span("hello world")},
									},
									Range: span("hello world"),
								},
								Range: span("hello world"),
							},
							Range: span("hello world"),
						},
					},
					Range: enclosing(span("--xyz"), span("--xyz--")),
				},
				Range: enclosing(span("### upload"), span("--xyz--")),
			},
			ast.Request{
				Name:   &ast.Ident{Range: span("find-users")},
				Method: ast.Ident{Range: spanAfter("### find-users", "POST")},
				URL: ast.TextLiteral{
					Value: "https://example.com/find",
					Range: span("https://example.com/find"),
				},
				Headers: []ast.Header{
					{
						Name: ast.Ident{Range: spanAfter("### find-users", "Content-Type")},
						Value: ast.TextLiteral{
							Value: "application/x-www-form-urlencoded",
							Range: span("application/x-www-form-urlencoded"),
						},
						Range: span("Content-Type: application/x-www-form-urlencoded"),
					},
				},
				// A form encoded body with a templated field and a literal one.
				Body: ast.BodyForm{
					Fields: []ast.FormField{
						{
							Key: ast.TextLiteral{Value: "search", Range: span("search")},
							Value: ast.Interp{
								Inner: ast.Ident{Range: span("query")},
								Range: span("{{ query }}"),
							},
							Range: span("search={{ query }}"),
						},
						{
							Key:   ast.TextLiteral{Value: "sort", Range: span("sort")},
							Value: ast.TextLiteral{Value: "desc", Range: span("desc")},
							Range: span("sort=desc"),
						},
					},
					Range: span("search={{ query }}&sort=desc"),
				},
				Range: enclosing(span("### find-users"), span("search={{ query }}&sort=desc")),
			},
		},
		Range: source.Span{File: file, StartOffset: 0, EndOffset: len(src)},
	}
}
