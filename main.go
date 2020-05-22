package main

import (
	"log"
	"math/rand"

	"runtime"
	"time"
	"xlog"
	"proxy"
	"storage"
	"util"
)

var cfg proxy.ProxyConf
func main() {
	s := proxy.NewProxy(&cfg, nil)
	s.Start()
	select{}
}
