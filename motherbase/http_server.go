package motherbase

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/yangzhao28/phantom/commonlog"
)

var httpLogger = commonlog.NewLogger("httpserver", "log", commonlog.DEBUG)

var manager = NewAgentGateway()

func ListConfig(w http.ResponseWriter, req *http.Request) {
	httpLogger.Debug("receive getconfig request from", req.Host)
	itemList, err := manager.Cache.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		httpLogger.Warning(err.Error())
		return
	}
	body, err := json.Marshal(itemList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		httpLogger.Warning(err.Error())
		return
	}
	w.Write(body)
}

func GetConfig(w http.ResponseWriter, req *http.Request) {
	httpLogger.Debug("receive getconfig request from", req.Host)
	name := req.URL.Query().Get("name")
	if len(name) == 0 {
		http.Error(w, "missing 'name'", http.StatusBadRequest)
		httpLogger.Warning("missing 'name'")
		return
	}
	body, err := manager.Cache.Get(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		httpLogger.Warning(err.Error())
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
	httpLogger.Debug("receive doconfig request from", req.Host)
	name := req.URL.Query().Get("name")
	if len(name) == 0 {
		http.Error(w, "missing 'name'", http.StatusBadRequest)
		httpLogger.Warning("missing 'name'")
		return
	}
	if req.ContentLength == 0 {
		http.Error(w, "missing post 'body' for config "+name, http.StatusBadRequest)
		httpLogger.Warning("missing 'body'")
		return
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "can't read body: "+err.Error(), http.StatusInternalServerError)
		httpLogger.Warning("can't read body: " + err.Error())
		return
	}
	if err := manager.NewConfig(name, string(body)); err != nil {
		http.Error(w, "save failed: "+err.Error(), http.StatusInternalServerError)
		httpLogger.Warning("save failed: " + err.Error())
		return
	}
	httpLogger.Debug(fmt.Sprintf("name: %v, body: %s", name, body))
	httpLogger.Info(fmt.Sprintf("doConfig request done: name -- %v, body length -- %v", name, len(body)))
	io.WriteString(w, "done.\n")
}

func CreateServer() {
	go manager.Go()

	manager.NewAgent("localhost", 8611)

	if err := manager.Cache.Clean(); err != nil {
		httpLogger.Warning("something wrong when clean save files")
	}
	if err := manager.Cache.Reload(); err != nil {
		httpLogger.Warning("something wrong when clean save files")
	}
	http.HandleFunc("/listconfig", ListConfig)
	http.HandleFunc("/getconfig", GetConfig)
	http.HandleFunc("/doconfig", DoConfig)
	httpLogger.Notice("port: 12345")
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
