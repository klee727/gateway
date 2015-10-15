package gateway

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type HostConfig struct {
	Host string
	Port int
}

type Configure struct {
	LogDir         string
	SaveDir        string
	HttpService    HostConfig
	AgentInstances []HostConfig

	DetectRoundTimeBySecond int
	DifferRoundTimeBySecond int
}

func LoadConfig(fileName string) (*Configure, error) {
	configFile, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	// server port
	config, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}
	newConfig := &Configure{}
	if err := json.Unmarshal(config, newConfig); err != nil {
		return nil, err
	}
	return newConfig, nil
}
