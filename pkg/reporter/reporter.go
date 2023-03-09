// Copyright 2022 The KCL Authors. All rights reserved.

package reporter

import (
	"log"
	"os"
)

func init() {
	log.SetFlags(0)
}

// Report prints to the logger.
// Arguments are handled in the manner of fmt.Println.
func Report(v ...any) {
	log.Println(v...)
}

// ExitWithReport prints to the logger and exit with 0.
// Arguments are handled in the manner of fmt.Println.
func ExitWithReport(v ...any) {
	log.Println(v...)
	os.Exit(0)
}

// Fatal prints to the logger and exit with 1.
// Arguments are handled in the manner of fmt.Println.
func Fatal(v ...any) {
	log.Fatal(v...)
}
