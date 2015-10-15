package gateway

import "fmt"

type AgentGateway struct {
	Manager *AgentManager
}

func NewAgentGateway(config *Configure) *AgentGateway {
	manager := NewAgentManager(config)
	return &AgentGateway{
		Manager: manager,
	}
}

func (gateway *AgentGateway) NewAgent(host string, port int) error {
	var agent Configurable
	agent = NewBidderHttpBridge(host, port)
	name := fmt.Sprintf("%v:%v", host, port)
	gateway.Manager.AddAgent(name, &agent)
	return nil
}

func (gateway *AgentGateway) NewConfig(id string, config string) error {
	if err := gateway.Manager.ConfigureCache.Update(id, config); err != nil {
		return err
	}
	return nil
}

// http server

func (gateway *AgentGateway) Quit() {
	gateway.Manager.Quit()
}

func (gateway *AgentGateway) Go() {
	gateway.Manager.Go()
}
