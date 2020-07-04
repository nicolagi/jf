// Ignore in the build, this is meant to be used with go run.
// It will check that jf and jf9 give same outputs for a number of JSON documents.

// +build ignore

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/go-cmp/cmp"
)

var (
	jf  string
	jf9 string

	// A bounded queue for URLs to check.
	queue = make(chan string, 64)
	// Never enqueue twice.
	enqueued = make(map[string]struct{})

	compared   = 0
	matched    = 0
	queued     = 0
	overflowed = 0
	alreadySeen = 0
)

func main() {
	flag.StringVar(&jf, "jf", "jf", "`path` to first binary")
	flag.StringVar(&jf9, "jf9", "jf9", "`path` to second binary")
	initial := flag.String("initial", "https://swapi.dev/api/people/1", "initial URL to scrape")
	max := flag.Int("max", 64, "max number of documents fetched and compared")
	flag.Parse()
	offer(*initial)
	for compared < *max {
		select{
		case url := <-queue:
			compare(url)
			printStats()
		default:
			return
		}
	}
}

func compare(url string) {
	r, err := http.Get(url)
	if err != nil {
		log.Printf("error fetching %q, discarding: %v", url, err)
		return
	}
	defer r.Body.Close()
	var dup bytes.Buffer
	jfc := exec.Command(jf)
	jfc.Stdin = io.TeeReader(r.Body, &dup)
	jf9c := exec.Command(jf9)
	jf9c.Stdin = &dup
	jfout, jferr := jfc.CombinedOutput()
	jf9out, jf9err := jf9c.CombinedOutput()
	if jferr != nil {
		log.Printf("error running jf command, discarding: %v", jferr)
		return
	}
	if jf9err != nil {
		log.Printf("error running jf9 command, discarding: %v", jf9err)
		return
	}
	if diff := cmp.Diff(string(jfout), string(jf9out)); diff != "" {
		fmt.Printf("Difference for %q:\n%v", url, diff)
	} else {
		matched++
	}
	compared++
	addUrls(jfout)
}

func addUrls(output []byte) {
	s := bufio.NewScanner(bytes.NewReader(output))
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if strings.HasPrefix(fields[1], `"http`) {
			url := fields[1][1 : len(fields[1])-1]
			offer(url)
		}
	}
}

func offer(url string) {
	if _, ok := enqueued[url]; ok {
		alreadySeen++
		return
	}
	select {
	case queue <- url:
		enqueued[url] = struct{}{}
		queued++
	default:
		overflowed++
	}
}

func printStats() {
	fmt.Println(queued, compared, matched, overflowed, alreadySeen)
}
