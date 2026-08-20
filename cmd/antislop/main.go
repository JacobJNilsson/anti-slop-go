// Command antislop runs every anti-slop analyzer as a standalone
// multichecker. It also satisfies the go vet -vettool contract.
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	antislop "github.com/JacobJNilsson/anti-slop-go"
)

func main() { multichecker.Main(antislop.Analyzers()...) }
