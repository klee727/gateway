package motherbase

import (
	"fmt"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	manager := NewAgentManager(nil)
	fmt.Println("let's go")
	go manager.Go()
	time.Sleep(5)
	var instance Configurable
	instance = NewBidderHttpBridge("localhost", 8611)
	fmt.Println("add")
	manager.AddAgent("test", &instance)

	time.Sleep(5)
	fmt.Println("quit")
	manager.Quit()
}
