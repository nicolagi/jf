package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {
	many := flag.Bool("m", false, "decode many values")
	unbuffered := flag.Bool("u", false, "unbuffered (print output line by line)")
	flag.Parse()
	var opts []option
	if *many {
		opts = append(opts, acceptMany)
	}
	var out io.Writer
	if *unbuffered {
		out = os.Stdout
	} else {
		bio := bufio.NewWriter(os.Stdout)
		defer func() {
			if err := bio.Flush(); err != nil {
				// Is standard error going to work any better?
				log.Printf("Could not flush output: %v", err)
			}
		}()
		out = bio
	}
	f := newFlattener(os.Stdin, opts...)
	f.run(func(path string, value string, err error) {
		if err != nil {
			log.Printf("jf: %v", err)
			return
		}
		_, err = fmt.Fprintf(out, "%s\t%s\n", path, value)
		if err != nil {
			log.Printf("Could not write to output: %v", err)
			return
		}
	})
}
