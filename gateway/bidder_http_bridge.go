package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	HttpTimeout = 3 * time.Second
)

type BidderHttpBridge struct {
	Host string
	Port int
}

func NewBidderHttpBridge(host string, port int) *BidderHttpBridge {
	return &BidderHttpBridge{
		Host: host,
		Port: port,
	}
}

func (bridge *BidderHttpBridge) DoConfig(name string, body string) error {
	url := fmt.Sprintf("http://%v:%v/agent?agent_name=%v&agent_type=linear", bridge.Host, bridge.Port, name)
	request, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: HttpTimeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			body = []byte("no response")
		}
		return errors.New(fmt.Sprintf("Something wrong when doconfig %v on %v: %v", name, bridge.Host, string(body)))
	}
	return nil
}

func (bridge *BidderHttpBridge) UnConfig(name string) error {
	url := fmt.Sprintf("http://%v:%v/agent?agent_name=%v&agent_type=linear", bridge.Host, bridge.Port, name)
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: HttpTimeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			body = []byte("no response")
		}
		return errors.New(fmt.Sprintf("Something wrong when unconfig %v on %v: %v", name, bridge.Host, string(body)))
	}
	return nil
}

func (bridge *BidderHttpBridge) Ping() error {
	url := fmt.Sprintf("http://%v:%v/ping", bridge.Host, bridge.Port)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: HttpTimeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			body = []byte("no response")
		}
		return errors.New(fmt.Sprintf("Something wrong when ping %v: %v", bridge.Host, string(body)))
	}
	return nil
}

func (bridge *BidderHttpBridge) ListConfig() (map[string]string, error) {
	url := fmt.Sprintf("http://%v:%v/agents", bridge.Host, bridge.Port)
	request, err := http.NewRequest("GET", url, nil)
	items := make(map[string]string)
	if err != nil {
		return items, err
	}
	client := &http.Client{Timeout: HttpTimeout}
	response, err := client.Do(request)
	if err != nil {
		return items, err
	}
	if response.StatusCode != 200 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			body = []byte("no response")
		}
		return items, errors.New(fmt.Sprintf("Something wrong when listconfig on %v: %v", bridge.Host, string(body)))
	}
	body, err := ioutil.ReadAll(response.Body)
	decoded := make(map[string]interface{}, 0)
	if err = json.Unmarshal(body, &decoded); err != nil {
		return items, err
	}
	// build items
	itemCount := 0
	if countItem, ok := decoded["count"]; ok {
		if count, ok := countItem.(float64); ok {
			itemCount = int(count)
		} else {
			return items, errors.New(fmt.Sprintf("invaild format when listconfig on %v: count should be integer", bridge.Host))
		}
	} else {
		return items, errors.New(fmt.Sprintf("invalid format when listconfig on %v: missing count", bridge.Host))
	}
	if itemCount > 0 {
		if agents, ok := decoded["agents"]; ok {
			for _, agent := range agents.([]interface{}) {
				if name, ok := (agent.(map[string]interface{}))["name"]; ok {
					if md5sum, ok := (agent.(map[string]interface{}))["md5sum"]; ok {
						items[name.(string)] = md5sum.(string)
					}
				}
			}
		}
	}
	return items, nil
}
