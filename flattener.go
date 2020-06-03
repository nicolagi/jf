package main

import (
	"fmt"
	"io"
)

type pair struct {
	path  string
	value string
	err   error
}

var zeroPair = pair{}

type option func(*flattener)

func acceptMany(f *flattener) {
	f.many = true
}

// flattener is a recursive-descent parser that produces, instead of
// a parse tree, a sequence of pathname-value pairs. Example:
//
// 	; curl -sL https://swapi.dev/api/people/2/ | jf
// 	.	{}
// 	."name"	"C-3PO"
// 	."height"	"167"
// 	."mass"	"75"
// 	."hair_color"	"n/a"
// 	."skin_color"	"gold"
// 	."eye_color"	"yellow"
// 	."birth_year"	"112BBY"
// 	."gender"	"n/a"
// 	."homeworld"	"http://swapi.dev/api/planets/1/"
// 	."films"	[]
// 	."films"[0]	"http://swapi.dev/api/films/1/"
// 	."films"[1]	"http://swapi.dev/api/films/2/"
// 	."films"[2]	"http://swapi.dev/api/films/3/"
// 	."films"[3]	"http://swapi.dev/api/films/4/"
// 	."films"[4]	"http://swapi.dev/api/films/5/"
// 	."films"[5]	"http://swapi.dev/api/films/6/"
// 	."species"	[]
// 	."species"[0]	"http://swapi.dev/api/species/2/"
// 	."vehicles"	[]
// 	."starships"	[]
// 	."created"	"2014-12-10T15:10:51.357000Z"
// 	."edited"	"2014-12-20T21:17:50.309000Z"
// 	."url"	"http://swapi.dev/api/people/2/"
//
// Be sure to consume all pairs using nextPair until io.EOF in
// order not to leak a goroutine.
//
// Note that from a basic benchmark this implementation is twice as
// slow as one based on standard library's json.Unmarshal, but the
// point of this exercise was learning to write a lexer and a parser
// by hand, with a focus on (my notion of) readability.
//
// goos: netbsd
// goarch: amd64
// BenchmarkOwnFlattener-8   	   12930	     94343 ns/op	    5939 B/op	      82 allocs/op
// BenchmarkOtherFlattener-8   	   22353	     54723 ns/op	   10178 B/op	     164 allocs/op
//
// Indeed, I did write a version based on json.Unmarshal to compare
// the two implementations (using testing/quick) to try and ensure
// correctness. That implementation just reads up all the input,
// passes it as byte slice to json.Unmarshal, and use reflection to
// navigate the generated value and generate the list of
// pathname-value pairs.
//
// An interesting property of the present implementation is that the
// output order is determined by the order of the input, contrary to
// the json.Unmarshal-based implementation.
type flattener struct {
	l      *lexer
	last   item // Last item got from the lexer.
	repeat bool // Whether fetching the next item returns last or reads a new one from the lexer.
	pairs  chan pair
	many   bool // Decode only one value or many?
}

func newFlattener(r io.Reader, opts ...option) *flattener {
	f := &flattener{
		l:     newLexer(r),
		pairs: make(chan pair),
	}
	for _, o := range opts {
		o(f)
	}
	go func() {
		for {
			// Don't care for the return value here, that's only used to
			// interrupt the recursive descent. Any error will get to the
			// consumer via NextPair.
			_ = f.flattenValue(".")
			if it := f.nextItem(); it.typ == itemEOF {
				break
			} else if !f.many {
				_ = f.errorf("expected to flatten one value and get EOF, got: %v", it.val)
				break
			} else {
				f.backup()
			}
		}
		close(f.pairs)
	}()
	return f
}

// nextPair can be used to iterate over the flattened properties.
// The error is io.EOF when there are no more properties.
func (f *flattener) nextPair() (path string, value string, err error) {
	p := <-f.pairs
	if p == zeroPair {
		return "", "", io.EOF
	}
	return p.path, p.value, p.err
}

func (f *flattener) nextItem() item {
	if !f.repeat {
		f.last = f.l.nextItem()
	} else {
		f.repeat = false
	}
	return f.last
}

// Only call once per call to nextItem.
func (f *flattener) backup() {
	f.repeat = true
}

func (f *flattener) errorf(format string, a ...interface{}) (errored bool) {
	f.pairs <- pair{err: fmt.Errorf(format, a...)}
	return true
}

func (f *flattener) flattenValue(path string) (errored bool) {
	switch it := f.nextItem(); it.typ {
	case itemError:
		return f.errorf("flattenValue: lexer error: %v", it.val)
	case itemLeftCurlyBrace:
		f.backup()
		f.pairs <- pair{path: path, value: `{}`}
		return f.flattenObject(path)
	case itemLeftBracket:
		f.backup()
		f.pairs <- pair{path: path, value: `[]`}
		return f.flattenArray(path)
	case itemQuotedString, itemUnquotedString:
		f.pairs <- pair{path: path, value: it.val}
		return false
	default:
		return f.errorf("flattenValue: unexpected lexeme: %v", it)
	}
}

func (f *flattener) flattenObject(path string) (errored bool) {
	f.nextItem()
	if f.nextItem().typ == itemRightCurlyBrace {
		return false
	}
	f.backup()
	for {
		it := f.nextItem()
		if it.typ != itemQuotedString {
			return f.errorf("flattenObject: expected quoted string for key, got: %v", it)
		}
		if it := f.nextItem(); it.typ != itemColon {
			return f.errorf("flattenObject: expected colon after key, got: %v", it)
		}
		var child string
		if path == "." {
			child = path + it.val
		} else {
			child = fmt.Sprintf("%s.%s", path, it.val)
		}
		if f.flattenValue(child) {
			return true
		}
		// Either the object is complete, or there's a comma and another key-value pair.
		it = f.nextItem()
		if it.typ == itemRightCurlyBrace {
			return false
		}
		if it.typ != itemComma {
			return f.errorf("flattenObject: expected comma or right curly brace after key-value pair, got: %v", it)
		}
	}
}

func (f *flattener) flattenArray(path string) (errored bool) {
	f.nextItem()
	if f.nextItem().typ == itemRightBracket {
		return false
	}
	f.backup()
	for index := 0; ; index++ {
		if f.flattenValue(fmt.Sprintf("%s[%d]", path, index)) {
			return true
		}
		// Either the array is complete, or there's a comma and another value.
		it := f.nextItem()
		if it.typ == itemRightBracket {
			return false
		}
		if it.typ != itemComma {
			return f.errorf("flattenArray: expected comma or right bracket after value, got: %v", it)
		}
	}
}
