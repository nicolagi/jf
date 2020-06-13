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

type option func(*flattener)

func acceptMany(f *flattener) {
	f.many = true
}

// flattener is a recursive-descent parser that produces, instead of
// a parse tree, a sequence of pathname-value pairs. Example:
//
// 	; curl -sL https://api.spacexdata.com/v3/launches/latest | jf | grep links
// 	."links"	{}
// 	."links"."mission_patch"	"https://images2.imgbox.com/d2/3b/bQaWiil0_o.png"
// 	."links"."mission_patch_small"	"https://images2.imgbox.com/9a/96/nLppz9HW_o.png"
// 	."links"."reddit_campaign"	"https://www.reddit.com/r/spacex/comments/gwbr4t/starlink8_launch_campaign_thread/"
// 	."links"."reddit_launch"	"https://www.reddit.com/r/spacex/comments/h7gqlc/rspacex_starlink_8_official_launch_discussion/"
// 	."links"."reddit_recovery"	null
// 	."links"."reddit_media"	"https://www.reddit.com/r/spacex/comments/h842qk/rspacex_starlink8_media_thread_photographer/"
// 	."links"."presskit"	null
// 	."links"."article_link"	null
// 	."links"."wikipedia"	"https://en.wikipedia.org/wiki/Starlink"
// 	."links"."video_link"	"https://youtu.be/8riKQXChPGg"
// 	."links"."youtube_id"	"8riKQXChPGg"
// 	."links"."flickr_images"	[]
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
	many   bool // Decode only one value or many?
	cb     func(path string, value string, err error)
}

func newFlattener(r io.Reader, opts ...option) *flattener {
	f := &flattener{
		l: newLexer(r),
	}
	for _, o := range opts {
		o(f)
	}
	return f
}

func (f *flattener) run(cb func(path string, value string, err error)) {
	f.cb = cb
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

}

// For convenience in unit tests.
func (f *flattener) collect() (output []pair) {
	f.run(func(path string, value string, err error) {
		output = append(output, pair{path: path, value: value, err: err})
	})
	return
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
	f.cb("", "", fmt.Errorf(format, a...))
	return true
}

func (f *flattener) flattenValue(path string) (errored bool) {
	switch it := f.nextItem(); it.typ {
	case itemError:
		return f.errorf("flattenValue: lexer error: %v", it.val)
	case itemLeftCurlyBrace:
		f.backup()
		f.cb(path, "{}", nil)
		return f.flattenObject(path)
	case itemLeftBracket:
		f.backup()
		f.cb(path, "[]", nil)
		return f.flattenArray(path)
	case itemQuotedString, itemUnquotedString:
		f.cb(path, it.val, nil)
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
