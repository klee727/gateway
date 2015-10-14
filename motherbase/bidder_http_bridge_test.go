package motherbase

import (
	"fmt"
	"testing"
)

func TestPing(t *testing.T) {
	bridge := BidderHttpBridge{
		Host: "localhost",
		Port: 8611,
	}
	fmt.Println("ping")
	fmt.Println(bridge.Ping())
	fmt.Println("done")
}

func TestListConfig(t *testing.T) {
	bridge := BidderHttpBridge{
		Host: "localhost",
		Port: 8611,
	}
	fmt.Println("list")
	fmt.Println(bridge.ListConfig())
	fmt.Println("done")
}
