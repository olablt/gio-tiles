package main

import (
	"log"
	"os"
)

var (
// dataLog   *log.Logger
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	// out := io.MultiWriter(os.Stdout)
	// dataLog = log.New(out, "DATA: ", log.Ldate|log.Ltime|log.Lshortfile)
}
