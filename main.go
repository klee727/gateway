package main

import (
	"github.com/yangzhao28/phantom/commonlog"
	"github.com/yangzhao28/phantom/motherbase"
)

var logger = commonlog.NewLogger("main", "log", commonlog.DEBUG)

func main() {
	logger.Notice("service start")
	motherbase.CreateServer()
}
