# gopar

A [Parsec](!http://hackage.haskell.org/package/parsec-3.0.0)-like library for Go.

Inspaired by https://github.com/sanyaade-buildtools/goparsec

To know more about theory of parser combinators, refer to : http://en.wikipedia.org/wiki/Parser_combinator

## Example

```go
package main

import (
	"fmt"
	. "parsec"
)

func main() {
	cell := Many(NoneOf([]byte(",\n"))).ToString()
	line := cell.SepBy(Char(','))
	csv := line.SepBy(Char('\n'))

	ast, err := csv.Parse("foo,bar\nhello,world,123")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("%+v\n", ast)
}

```
