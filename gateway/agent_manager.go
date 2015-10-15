package gateway

import (
	"strings"
	"sync"
	"time"

	"github.com/op/go-logging"
	"github.com/yangzhao28/gateway/commonlog"
)

// 收取请求(http)        √
// cache, 本地存储       √
// Agent管理             √
// 对Agent 3 routine:
//   - 配置              √
//   - 探活              √
//   - diff              √

type AgentManager struct {
	enableAgents map[string]Configurable
	enableMutex  sync.RWMutex

	availableAgents map[string]Configurable
	availableMutex  sync.RWMutex

	agentEnableChannel  chan *AgentEvent
	agentDisablechannel chan *AgentEvent
	agentCreateChannel  chan *AgentEvent
	agentRemoveChannel  chan *AgentEvent

	agentRunDetector    chan int
	agentRunDiffer      chan int
	detectorRoundSecond time.Duration
	differRoundSecond   time.Duration

	ConfigureCache        *PersistCache
	loggerForAgentManager *logging.Logger

	syncer sync.WaitGroup
	quit   chan bool
}

type AgentEvent struct {
	name         string
	configurable *Configurable
}

func NewAgentManager(config *Configure) *AgentManager {
	manager := &AgentManager{
		enableAgents:    make(map[string]Configurable),
		availableAgents: make(map[string]Configurable),

		agentEnableChannel:  make(chan *AgentEvent),
		agentDisablechannel: make(chan *AgentEvent),
		agentCreateChannel:  make(chan *AgentEvent),
		agentRemoveChannel:  make(chan *AgentEvent),

		agentRunDetector: make(chan int),
		agentRunDiffer:   make(chan int),

		detectorRoundSecond: time.Duration(config.DetectRoundTimeBySecond) * time.Second,
		differRoundSecond:   time.Duration(config.DifferRoundTimeBySecond) * time.Second,

		ConfigureCache:        NewPersistCache(config.SaveDir),
		loggerForAgentManager: commonlog.NewLogger("agentmanager", config.LogDir, commonlog.DEBUG),

		quit: make(chan bool),
	}
	return manager
}

func (manager *AgentManager) enableAgent(name string, instance *Configurable) {
	manager.enableMutex.Lock()
	defer manager.enableMutex.Unlock()

	if _, ok := manager.enableAgents[name]; !ok {
		manager.enableAgents[name] = *instance
		manager.loggerForAgentManager.Debug("new enable agent " + name)
	}
}

func (manager *AgentManager) disableAgent(name string) {
	manager.enableMutex.Lock()
	defer manager.enableMutex.Unlock()
	if _, ok := manager.enableAgents[name]; ok {
		delete(manager.enableAgents, name)
		manager.loggerForAgentManager.Debug("disable agent %v", name)
	}
}

func (manager *AgentManager) NewAvailableAgent(name string, instance *Configurable) {
	manager.availableMutex.Lock()
	defer manager.availableMutex.Unlock()
	if _, ok := manager.availableAgents[name]; !ok {
		manager.availableAgents[name] = *instance
		manager.loggerForAgentManager.Debug("new available agent %v", name)
	}
}

func (manager *AgentManager) RemoveAvailableAgent(name string) {
	manager.availableMutex.Lock()
	defer manager.availableMutex.Unlock()
	if _, ok := manager.availableAgents[name]; ok {
		delete(manager.availableAgents, name)
		manager.loggerForAgentManager.Debug("delete agent %v", name)
	}
}

func (manager *AgentManager) Controller() {
	manager.loggerForAgentManager.Info("controller service start.")
	defer manager.loggerForAgentManager.Info("controller service shutdown.")

	defer manager.syncer.Done()
	for {
		select {
		case event := <-manager.agentEnableChannel:
			manager.enableAgent(event.name, event.configurable)
		case event := <-manager.agentDisablechannel:
			manager.disableAgent(event.name)
		case event := <-manager.agentCreateChannel:
			manager.NewAvailableAgent(event.name, event.configurable)
		case event := <-manager.agentRemoveChannel:
			// TODO 没有这么简单， 需要首先关掉所有 Configure，再去除Agent
			manager.RemoveAvailableAgent(event.name)
			manager.disableAgent(event.name)
		case <-manager.quit:
			break
		}
	}
}

func (manager *AgentManager) Detector() {
	manager.loggerForAgentManager.Info("detector service start.")
	defer manager.loggerForAgentManager.Info("detector service shutdown.")

	defer manager.syncer.Done()
	for {
		select {
		case <-manager.agentRunDetector:
			waitForDone := sync.WaitGroup{}
			func() {
				manager.availableMutex.RLock()
				defer manager.availableMutex.RUnlock()
				for name, agent := range manager.availableAgents {
					waitForDone.Add(1)
					go func(name string, bridge Configurable) {
						defer waitForDone.Done()
						if bridge.Ping() != nil {
							manager.agentDisablechannel <- &AgentEvent{name, &agent}
						} else {
							manager.agentEnableChannel <- &AgentEvent{name, &agent}
						}
					}(name, agent)
				}
			}()
			waitForDone.Wait()
		case <-manager.quit:
			break
		}
	}
}

func (manager *AgentManager) Scheduler() {
	manager.loggerForAgentManager.Info("scheduler service start.")
	defer manager.loggerForAgentManager.Info("scheduler service shutdown.")
	defer manager.syncer.Done()

	detectorTimer := time.NewTimer(manager.detectorRoundSecond)
	differTimer := time.NewTimer(manager.differRoundSecond)
	for {
		select {
		case <-detectorTimer.C:
			manager.agentRunDetector <- 0
			detectorTimer.Reset(manager.detectorRoundSecond)
		case <-differTimer.C:
			manager.agentRunDiffer <- 0
			differTimer.Reset(manager.differRoundSecond)
		case <-manager.quit:
			break
		}
	}
}

func (manager *AgentManager) Differ() {
	manager.loggerForAgentManager.Info("differ serivce start.")
	defer manager.loggerForAgentManager.Info("differ service shutdown.")

	defer manager.syncer.Done()
	for {
		select {
		case <-manager.agentRunDiffer:
			if len(manager.enableAgents) == 0 {
				manager.loggerForAgentManager.Debug("no activated agents")
				continue
			}
			func() {
				// get current agent list, eg: activated
				activatedAgents := manager.ConfigureCache.List()
				manager.loggerForAgentManager.Info("expect %v agents", len(activatedAgents))
				// use to filter found agents
				activatedAgentsId := make(map[string]bool)
				for _, value := range activatedAgents {
					activatedAgentsId[value.id] = true
				}
				waitForDone := sync.WaitGroup{}
				func() {
					manager.availableMutex.RLock()
					defer manager.availableMutex.RUnlock()
					for name, agent := range manager.enableAgents {
						waitForDone.Add(1)
						manager.loggerForAgentManager.Debug("diff -> " + name)
						go func(name string, bridge Configurable) {
							defer waitForDone.Done()
							// map[string]string
							foundAgents, err := bridge.ListConfig()
							if err != nil {
								// do something
								manager.loggerForAgentManager.Warning("fail to list config on %v: %v", name, err.Error())
								return
							}
							manager.loggerForAgentManager.Info("found %v agents on %v", len(foundAgents), name)
							// check unexpected agents
							for id, _ := range foundAgents {
								if _, ok := activatedAgentsId[id]; !ok {
									agent.UnConfig(id)
								}
							}
							// check agent not updated
							for _, info := range activatedAgents {
								if md5sum, ok := foundAgents[info.id]; ok && strings.ToLower(md5sum) == strings.ToLower(info.md5sum) {
									continue
								}
								manager.loggerForAgentManager.Warning("expect id %v on %v but missed", info.id, name)
								go func(configId string) {
									manager.loggerForAgentManager.Debug("try reconfigure agent: %v", configId)
									if manager.ConfigureCache != nil {
										jsonConfig, err := manager.ConfigureCache.Get(configId)
										if err != nil {
											manager.loggerForAgentManager.Warning("no config for: %v", configId)
											return
										}
										manager.loggerForAgentManager.Debug("do config on %v", name)
										bridge.DoConfig(configId, jsonConfig)
									}
								}(info.id)
							}
						}(name, agent)
					}
				}()
				waitForDone.Wait()
			}()
		case <-manager.quit:
			break
		}
	}
}

func (manager *AgentManager) Quit() {
	manager.loggerForAgentManager.Debug("Quit")
	close(manager.quit)
}

func (manager *AgentManager) AddAgent(name string, instance *Configurable) {
	manager.loggerForAgentManager.Debug("add agent %v", name)
	manager.agentCreateChannel <- &AgentEvent{
		name:         name,
		configurable: instance,
	}
}

func (manager *AgentManager) RemoveAgent(name string, instance *Configurable) {
	manager.loggerForAgentManager.Debug("remove agent %v", name)
	manager.agentRemoveChannel <- &AgentEvent{
		name:         name,
		configurable: instance,
	}
}

func (manager *AgentManager) Go() error {
	if err := manager.ConfigureCache.CleanInvalidFiles(); err != nil {
		return err
	}
	if err := manager.ConfigureCache.Reload(); err != nil {
		return err
	}

	for _, value := range manager.ConfigureCache.List() {
		manager.loggerForAgentManager.Debug("found config item id:%v md5:%v time:%v", value.id, value.md5sum, value.updateTime)
	}

	manager.syncer.Add(1)
	go manager.Controller()
	manager.syncer.Add(1)
	go manager.Detector()
	manager.syncer.Add(1)
	go manager.Scheduler()
	manager.syncer.Add(1)
	go manager.Differ()

	manager.loggerForAgentManager.Info("agent manager service started")
	manager.syncer.Wait()
	return nil
}
