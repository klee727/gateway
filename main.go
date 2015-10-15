package main

import (
	"fmt"
	"os"

	"github.com/yangzhao28/gateway/gateway"
)

func main() {
	config, err := gateway.LoadConfig("conf/gateway.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't read or parse config file: ", err.Error())
	}
	gateway.StartService(config)
}
