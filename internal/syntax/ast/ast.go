// Package ast provides the abstract syntax tree for .http files.
package ast

import (
	"go.followtheprocess.codes/arc/internal/syntax/source"
)

// Node is the interface implemented by all AST nodes.
type Node interface {
	// Pos returns the [source.Span] containing the node in the original
	// source text.
	Pos() source.Span
}

// Statement is the interface implemented by all statement AST nodes.
type Statement interface {
	Node

	statementNode()
}

// Expression is the interface implemented by all expression AST nodes.
type Expression interface {
	Node

	expressionNode()
}
