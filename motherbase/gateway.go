package motherbase

import "fmt"

type AgentGateway struct {
	Cache   *PersistCache
	Manager *AgentManager
}

func NewAgentGateway() *AgentGateway {
	cache := NewPersistCache("save")
	manager := NewAgentManager(cache)

	return &AgentGateway{
		Cache:   cache,
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
	if err := gateway.Cache.Replace(id, config); err != nil {
		return err
	}
	return nil
}

func (gateway *AgentGateway) Quit() {
	gateway.Manager.Quit()
}

func (gateway *AgentGateway) Go() {
	gateway.Manager.Go()
}
