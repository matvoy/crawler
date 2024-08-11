package main

import (
	"log"
	"time"
)

const (
	baseURL = "https://monzo.com"
)

func main() {
	var p Parser
	if err := p.Init(); err != nil {
		log.Fatalf("error creating file: %s", err)
	}
	log.Printf("Start parsing %s/\n", baseURL)
	start := time.Now()
	p.Parse(baseURL)
	end := time.Since(start)
	log.Printf("End. Count: %v; Time: %v;\n", p.count, end)
}
