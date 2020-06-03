package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type itemType int

const (
	itemError itemType = iota
	itemEOF

	itemColon
	itemComma

	itemLeftBracket
	itemRightBracket
	itemLeftCurlyBrace
	itemRightCurlyBrace

	// For the purpose of flattening JSON documents, we don't have to be
	// precise in how we break up primitive values. The unquoted string
	// item type will apply to numbers, true, false, null, and also
	// illegal strings.
	itemQuotedString
	itemUnquotedString
)

const eof rune = -1

type item struct {
	typ itemType
	val string
}

// String implements fmt.Stringer.
func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	if len(i.val) > 10 {
		return fmt.Sprintf("%.10q...", i.val)
	}
	return fmt.Sprintf("%q", i.val)
}

type stateFn func(*lexer) stateFn

// lexer is a lexer for JSON input used in the flattening parser (see
// flattener.go). Could use the standard library but I'm doing this
// as an exercise along while viewing
// https://invidio.us/watch?v=HxaD_trXwRE. This is a variation on the
// technique in the video, in that this lexer runs with a stream as
// an input, not a string containing the whole document.
type lexer struct {
	input  *bufio.Reader
	buffer bytes.Buffer
	width  int // The width of last rune read from input and written to the buffer.
	items  chan item
	state  stateFn
}

func newLexer(r io.Reader) *lexer {
	bio, ok := r.(*bufio.Reader)
	if !ok {
		bio = bufio.NewReader(r)
	}
	l := &lexer{
		input: bio,
		items: make(chan item, 1),
		state: lexWhitespace,
	}
	return l
}

func (l *lexer) nextItem() item {
	for {
		select {
		case it := <-l.items:
			return it
		default:
			if l.state == nil {
				l.items <- item{typ: itemEOF}
			} else {
				l.state = l.state(l)
			}
		}
	}
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.buffer.String()}
	l.buffer.Reset()
}

func (l *lexer) next() (r rune) {
	var err error
	r, l.width, err = l.input.ReadRune()
	if err != nil {
		l.width = 0
		return eof
	}
	l.buffer.WriteRune(r)
	return r
}

func (l *lexer) ignore() {
	l.buffer.Reset()
}

// Can be called only once per call of next.
func (l *lexer) backup() {
	if l.width > 0 {
		// An error would be returned if ReadRune wasn't the previous
		// operation on l.input.
		_ = l.input.UnreadRune()
		l.buffer.Truncate(l.buffer.Len() - l.width)
	}
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) accept(valid string) bool {
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) errorf(format string, a ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, a...),
	}
	return nil
}

func lexWhitespace(l *lexer) stateFn {
	l.acceptRun(" \t\r\n")
	l.ignore()

	switch l.peek() {
	case eof:
		return nil
	case ':':
		return lexColon
	case ',':
		return lexComma
	case '{':
		return lexLeftCurlyBrace
	case '[':
		return lexLeftBracket
	case '}':
		return lexRightCurlyBrace
	case ']':
		return lexRightBracket
	case '"':
		return lexQuotedString
	default:
		return lexUnquotedString
	}
}

func lexColon(l *lexer) stateFn {
	l.accept(":")
	l.emit(itemColon)
	return lexWhitespace
}

func lexComma(l *lexer) stateFn {
	l.accept(",")
	l.emit(itemComma)
	return lexWhitespace
}

func lexLeftBracket(l *lexer) stateFn {
	l.accept("[")
	l.emit(itemLeftBracket)
	return lexWhitespace
}

func lexRightBracket(l *lexer) stateFn {
	l.accept("]")
	l.emit(itemRightBracket)
	return lexWhitespace
}

func lexLeftCurlyBrace(l *lexer) stateFn {
	l.accept("{")
	l.emit(itemLeftCurlyBrace)
	return lexWhitespace
}

func lexRightCurlyBrace(l *lexer) stateFn {
	l.accept("}")
	l.emit(itemRightCurlyBrace)
	return lexWhitespace
}

func lexQuotedString(l *lexer) stateFn {
	l.accept(`"`)
	for {
		switch l.next() {
		case '\\':
			l.next()
		case '"':
			l.emit(itemQuotedString)
			return lexWhitespace
		case eof:
			return l.errorf("unfinished quoted string")
		}
	}
}

// Any piece of garbage delimited by whitespace or special characters
// is lexed here.
func lexUnquotedString(l *lexer) stateFn {
	for {
		r := l.next()
		if strings.ContainsRune(" \t\r\n{}[]:,\"", r) || r == eof {
			l.backup()
			l.emit(itemUnquotedString)
			return lexWhitespace
		}
	}
}
