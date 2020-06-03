package main

import (
	"strings"
	"testing"
)

type lexerTest struct {
	input  string
	output []item
}

// The main testing is via conformance checking in flattener_test.go.
// Here we'll only add a few basic tests and more tests for specific
// regressions as they are encountered.
func TestLexerRegressions(t *testing.T) {
	tests := []lexerTest{
		{
			input: "{}",
			output: []item{
				{typ: itemLeftCurlyBrace, val: "{"},
				{typ: itemRightCurlyBrace, val: "}"},
				{typ: itemEOF},
			},
		},
		{
			input: "[]",
			output: []item{
				{typ: itemLeftBracket, val: "["},
				{typ: itemRightBracket, val: "]"},
				{typ: itemEOF},
			},
		},
		{
			input: "unquoted",
			output: []item{
				{typ: itemUnquotedString, val: "unquoted"},
				{typ: itemEOF},
			},
		},
		{
			input: "unquoted string",
			output: []item{
				{typ: itemUnquotedString, val: "unquoted"},
				{typ: itemUnquotedString, val: "string"},
				{typ: itemEOF},
			},
		},
		{
			input: "42",
			output: []item{
				{typ: itemUnquotedString, val: "42"},
				{typ: itemEOF},
			},
		},
		{
			input: `""`,
			output: []item{
				{typ: itemQuotedString, val: `""`},
				{typ: itemEOF},
			},
		},
		{
			input: `"quoted string"`,
			output: []item{
				{typ: itemQuotedString, val: `"quoted string"`},
				{typ: itemEOF},
			},
		},
		{
			input: `"quoted string with \"escapes\""`,
			output: []item{
				{typ: itemQuotedString, val: `"quoted string with \"escapes\""`},
				{typ: itemEOF},
			},
		},
		{
			input: `{ "user": "zaphod", "age": 42 }`,
			output: []item{
				{typ: itemLeftCurlyBrace, val: "{"},
				{typ: itemQuotedString, val: `"user"`},
				{typ: itemColon, val: ":"},
				{typ: itemQuotedString, val: `"zaphod"`},
				{typ: itemComma, val: ","},
				{typ: itemQuotedString, val: `"age"`},
				{typ: itemColon, val: ":"},
				{typ: itemUnquotedString, val: "42"},
				{typ: itemRightCurlyBrace, val: "}"},
				{typ: itemEOF},
			},
		},
		{
			input: `[ 1, 1, 2, 3, 5, 8 ]`,
			output: []item{
				{typ: itemLeftBracket, val: "["},
				{typ: itemUnquotedString, val: "1"},
				{typ: itemComma, val: ","},
				{typ: itemUnquotedString, val: "1"},
				{typ: itemComma, val: ","},
				{typ: itemUnquotedString, val: "2"},
				{typ: itemComma, val: ","},
				{typ: itemUnquotedString, val: "3"},
				{typ: itemComma, val: ","},
				{typ: itemUnquotedString, val: "5"},
				{typ: itemComma, val: ","},
				{typ: itemUnquotedString, val: "8"},
				{typ: itemRightBracket, val: "]"},
				{typ: itemEOF},
			},
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			l := newLexer(strings.NewReader(tt.input))
			for _, want := range tt.output {
				it := l.nextItem()
				if got, want := it.typ, want.typ; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
				if got, want := it.val, want.val; got != want {
					t.Errorf("got %v, want %v", got, want)
				}
			}
		})
	}
}
