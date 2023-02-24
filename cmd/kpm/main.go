package main

import (
	"github.com/KusionStack/kpm/pkg/kpm"
	"os"
)

func main() {
	err := kpm.CLI(os.Args...)
	if err != nil {
		println(err.Error())
		return
	}
}
