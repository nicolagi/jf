// Command jf transforms a JSON element into a sequence of path-value
// pairs. Some examples follow. Flattening a single element:
//
// 	; curl -sL https://api.spacexdata.com/v3/capsules/C101 | jf
// 	.	{}
// 	."capsule_serial"	"C101"
// 	."capsule_id"	"dragon1"
// 	."status"	"retired"
// 	."original_launch"	"2010-12-08T15:43:00.000Z"
// 	."original_launch_unix"	1291822980
// 	."missions"	[]
// 	."missions"[0]	{}
// 	."missions"[0]."name"	"COTS 1"
// 	."missions"[0]."flight"	7
// 	."landings"	1
// 	."type"	"Dragon 1.0"
// 	."details"	"Reentered after three weeks in orbit"
// 	."reuse_count"	0
//
// By default only one element is accepted:
//
// 	; { curl -sL https://api.spacexdata.com/v3/capsules/C101 ; curl -sL https://api.spacexdata.com/v3/capsules/C102 } | jf
// 	2020/06/01 18:56:21 main: expected to flatten one value and get EOF, got: {
// 	<output for first JSON element>
//
// With -m, jflatten will flatten many JSON elements but paths may be
// duplicated as a result:
//
// 	; { curl -sL https://api.spacexdata.com/v3/capsules/C101 ; curl -sL https://api.spacexdata.com/v3/capsules/C102 } | jf -m | sort
// 	.	{}
// 	.	{}
// 	."capsule_id"	"dragon1"
// 	."capsule_id"	"dragon1"
// 	."capsule_serial"	"C101"
// 	."capsule_serial"	"C102"
// 	."details"	"First Dragon spacecraft"
// 	."details"	"Reentered after three weeks in orbit"
// 	."landings"	1
// 	."landings"	1
// 	."missions"	[]
// 	."missions"	[]
// 	."missions"[0]	{}
// 	."missions"[0]	{}
// 	."missions"[0]."flight"	7
// 	."missions"[0]."flight"	8
// 	."missions"[0]."name"	"COTS 1"
// 	."missions"[0]."name"	"COTS 2"
// 	."original_launch"	"2010-12-08T15:43:00.000Z"
// 	."original_launch"	"2012-05-22T07:44:00.000Z"
// 	."original_launch_unix"	1291822980
// 	."original_launch_unix"	1335944640
// 	."reuse_count"	0
// 	."reuse_count"	0
// 	."status"	"retired"
// 	."status"	"retired"
// 	."type"	"Dragon 1.0"
// 	."type"	"Dragon 1.0"
//
// It's easy to compose jf with the system diff to diff two JSON
// documents.
//
// 	bash-5.0$ diff -u <(curl -sL https://api.spacexdata.com/v3/capsules/C101 | jf) <(curl -sL https://api.spacexdata.com/v3/capsules/C102 | jf)
// 	--- /dev/fd/63	2020-05-18 00:00:01.743253402 +0100
// 	+++ /dev/fd/62	2020-05-18 00:00:01.743236779 +0100
// 	@@ -1,14 +1,14 @@
// 	 .	{}
// 	-."capsule_serial"	"C101"
// 	+."capsule_serial"	"C102"
// 	 ."capsule_id"	"dragon1"
// 	 ."status"	"retired"
// 	-."original_launch"	"2010-12-08T15:43:00.000Z"
// 	-."original_launch_unix"	1291822980
// 	+."original_launch"	"2012-05-22T07:44:00.000Z"
// 	+."original_launch_unix"	1335944640
// 	 ."missions"	[]
// 	 ."missions"[0]	{}
// 	-."missions"[0]."name"	"COTS 1"
// 	-."missions"[0]."flight"	7
// 	+."missions"[0]."name"	"COTS 2"
// 	+."missions"[0]."flight"	8
// 	 ."landings"	1
// 	 ."type"	"Dragon 1.0"
// 	-."details"	"Reentered after three weeks in orbit"
// 	+."details"	"First Dragon spacecraft"
// 	 ."reuse_count"	0
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
