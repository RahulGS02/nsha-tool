package main

import (
	"os"

	"github.com/rahul/nsha/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

