package main

import (
	"github.com/hagna/eforth"
	"os"
)

func main() {
	f := eforth.New(os.Stdout, os.Stdin)
	f.Main()
}
