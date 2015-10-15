package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/yangzhao28/gateway/commonlog"
)

var loggerForHttp = commonlog.NewLogger("http", "log", commonlog.DEBUG)
var manager *AgentGateway = nil

func ListConfig(w http.ResponseWriter, req *http.Request) {
	loggerForHttp.Debug("receive getconfig request from", req.Host)
	itemList := manager.Manager.ConfigureCache.List()
	body, err := json.Marshal(itemList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		loggerForHttp.Warning(err.Error())
		return
	}
	w.Write(body)
}

func GetConfig(w http.ResponseWriter, req *http.Request) {
	loggerForHttp.Debug("receive getconfig request from", req.Host)
	name := req.URL.Query().Get("name")
	if len(name) == 0 {
		http.Error(w, "missing 'name'", http.StatusBadRequest)
		loggerForHttp.Warning("missing 'name'")
		return
	}
	body, err := manager.Manager.ConfigureCache.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		loggerForHttp.Warning(err.Error())
		return
	}
	io.WriteString(w, body)
}

/**
* @brief eg: /doconfig?name=xxxxx
*
* @param http.ResponseWriter
* @param http.Request
*
* @return
 */
func DoConfig(w http.ResponseWriter, req *http.Request) {
	loggerForHttp.Debug("receive doconfig request from", req.Host)
	name := req.URL.Query().Get("name")
	if len(name) == 0 {
		http.Error(w, "missing 'name'", http.StatusBadRequest)
		loggerForHttp.Warning("missing 'name'")
		return
	}
	if req.ContentLength == 0 {
		http.Error(w, "missing post 'body' for config "+name, http.StatusBadRequest)
		loggerForHttp.Warning("missing 'body'")
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "can't read body: "+err.Error(), http.StatusInternalServerError)
		loggerForHttp.Warning("can't read body: " + err.Error())
		return
	}
	if err := manager.NewConfig(name, string(body)); err != nil {
		http.Error(w, "save failed: "+err.Error(), http.StatusInternalServerError)
		loggerForHttp.Warning("save failed: " + err.Error())
		return
	}
	loggerForHttp.Debug(fmt.Sprintf("name: %v, body: %s", name, body))
	loggerForHttp.Info(fmt.Sprintf("doConfig request done: name -- %v, body length -- %v", name, len(body)))
	io.WriteString(w, "done.\n")
}

func StartService(config *Configure) {
	manager := NewAgentGateway(config)
	go manager.Go()

	for _, value := range config.AgentInstances {
		manager.NewAgent(value.Host, value.Port)
		loggerForHttp.Info("add agent instance(%v:%v)", value.Host, value.Port)
	}

	http.HandleFunc("/listconfig", ListConfig)
	http.HandleFunc("/getconfig", GetConfig)
	http.HandleFunc("/doconfig", DoConfig)

	address := fmt.Sprintf("%v:%v", config.HttpService.Host, config.HttpService.Port)
	loggerForHttp.Info("http serivce started on %v", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
