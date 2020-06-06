// Command jf transforms a JSON element into a sequence of path-value
// pairs. Some examples follow. Flattening a single element:
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
// By default only one element is accepted:
//
// 	; { curl -sL https://swapi.dev/api/people/1/ ; curl -sL https://swapi.dev/api/people/2/ } | jf
// 	2020/06/01 18:56:21 main: expected to flatten one value and get EOF, got: {
// 	<output for first JSON element>
//
// With -m, jflatten will flatten many JSON elements but paths may be
// duplicated as a result:
//
// 	; { curl -sL https://swapi.dev/api/people/1/ ; curl -sL https://swapi.dev/api/people/2/ } | jf -m | sort
// 	.	{}
// 	.	{}
// 	."birth_year"	"112BBY"
// 	."birth_year"	"19BBY"
// 	."created"	"2014-12-09T13:50:51.644000Z"
// 	."created"	"2014-12-10T15:10:51.357000Z"
// 	."edited"	"2014-12-20T21:17:50.309000Z"
// 	."edited"	"2014-12-20T21:17:56.891000Z"
// 	."eye_color"	"blue"
// 	."eye_color"	"yellow"
// 	."films"	[]
// 	."films"	[]
// 	."films"[0]	"http://swapi.dev/api/films/1/"
// 	."films"[0]	"http://swapi.dev/api/films/1/"
// 	."films"[1]	"http://swapi.dev/api/films/2/"
// 	."films"[1]	"http://swapi.dev/api/films/2/"
// 	."films"[2]	"http://swapi.dev/api/films/3/"
// 	."films"[2]	"http://swapi.dev/api/films/3/"
// 	."films"[3]	"http://swapi.dev/api/films/4/"
// 	."films"[3]	"http://swapi.dev/api/films/6/"
// 	."films"[4]	"http://swapi.dev/api/films/5/"
// 	."films"[5]	"http://swapi.dev/api/films/6/"
// 	."gender"	"male"
// 	."gender"	"n/a"
// 	."hair_color"	"blond"
// 	."hair_color"	"n/a"
// 	."height"	"167"
// 	."height"	"172"
// 	."homeworld"	"http://swapi.dev/api/planets/1/"
// 	."homeworld"	"http://swapi.dev/api/planets/1/"
// 	."mass"	"75"
// 	."mass"	"77"
// 	."name"	"C-3PO"
// 	."name"	"Luke Skywalker"
// 	."skin_color"	"fair"
// 	."skin_color"	"gold"
// 	."species"	[]
// 	."species"	[]
// 	."species"[0]	"http://swapi.dev/api/species/2/"
// 	."starships"	[]
// 	."starships"	[]
// 	."starships"[0]	"http://swapi.dev/api/starships/12/"
// 	."starships"[1]	"http://swapi.dev/api/starships/22/"
// 	."url"	"http://swapi.dev/api/people/1/"
// 	."url"	"http://swapi.dev/api/people/2/"
// 	."vehicles"	[]
// 	."vehicles"	[]
// 	."vehicles"[0]	"http://swapi.dev/api/vehicles/14/"
// 	."vehicles"[1]	"http://swapi.dev/api/vehicles/30/"
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
