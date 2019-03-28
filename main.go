package main

import (
	"bicycle/spidy"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	logerr = log.New(os.Stderr, "", 0)
)

const (
	// 1 MB limit
	MaxBodySize = 1 * 1024 * 1024
)

type Config struct {
	maxDepth uint64
	reqPerSec uint64
	targetURI string
}

func NewConfig(maxDepth, reqPerSec uint64, targetURL string) *Config {
	cfg := new(Config)
	cfg.maxDepth = maxDepth
	cfg.reqPerSec = reqPerSec
	cfg.targetURI = targetURL
	return cfg
}

func parseArgs() (*Config, error){
	var maxDepth, reqPerSec uint64
	var targetURI string

	flag.Uint64Var(&maxDepth, "d", 2, "Maximum depth, 0 - no limit")
	flag.Uint64Var(&reqPerSec, "r", 5, "Requests per second, 0 - no limit")
	flag.StringVar(&targetURI, "u", "", "Target Uri, required")
	flag.Parse()

	if(targetURI == ""){
		return nil, errors.New("parseArgs: Target URI is required")
	}

	cfg := NewConfig(maxDepth, reqPerSec, targetURI)
	return cfg, nil
}

func check(err error) {
	if err != nil {
		logerr.Fatal(err)
	}
}

func main() {
	cfg, err := parseArgs()
	check(err)

	logerr.Println("[+] Spidy running")

	spidy, err := spidy.New(cfg.targetURI, int(cfg.maxDepth), MaxBodySize, int(cfg.reqPerSec))
	check(err)

	links := spidy.Run()
	for link := range links {
		fmt.Println(link)
	}

	logerr.Println("[+] Spidy finished")
}