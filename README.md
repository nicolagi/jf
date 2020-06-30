# jf — JSON flattener

Command jf is a JSON flattener, which composes well with existing system tools, e.g., with grep and diff; jf transforms a JSON element into a sequence of path-value pairs, one per row, tab-separated.

	; echo '{"fruit":[{"name":"banana"},{"name":"apple"}]}' | jf | tab
	.                  {}
	."fruit"           []
	."fruit"[0]        {}
	."fruit"[0]."name" "banana"
	."fruit"[1]        {}
	."fruit"[1]."name" "apple"

Installation: `go get -u github.com/nicolagi/jf`.

Documentation: https://pkg.go.dev/github.com/nicolagi/jf?tab=doc.
