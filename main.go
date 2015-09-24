package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ParallelRequest struct {
	id          int64
	hosts       []string
	httpRequest http.Request
}

type ParallelResponse struct {
	id           int64
	host         string
	code         int
	content      []byte
	httpResponse *http.Response
}

func GlobalIdService() chan int64 {
	id := make(chan int64)
	go func() {
		var indicer int64 = 0
		for {
			id <- indicer
			indicer++
		}
	}()
	return id
}

var Hosts []string = []string{
	//"http://www.baidu.com",
	//"http://www.jd.com",
	//"http://www.taobao.com",
	"http://localhost:20035",
	"http://localhost:20034",
}

func NewParallelRequest(req *http.Request) *ParallelRequest {
	newId := GlobalIdService()
	request := ParallelRequest{<-newId, Hosts, *req}
	return &request
}

func Do(request *ParallelRequest, globalTimeout time.Duration) (http.Response, string) {
	resultQueue := make(chan ParallelResponse)
	body, _ := ioutil.ReadAll(request.httpRequest.Body)
	for index, _ := range request.hosts {
		go func(address string, httpRequest http.Request) {
			httpClient := &http.Client{
				Transport: &http.Transport{
					Dial: func(netw, addr string) (net.Conn, error) {
						c, err := net.DialTimeout(netw, addr, globalTimeout)
						if err != nil {
							return nil, err
						}
						deadline := time.Now().Add(2 * globalTimeout)
						c.SetDeadline(deadline)
						return c, nil
					},
					ResponseHeaderTimeout: globalTimeout,
				},
			}
			// 转发 Request
			rebuildRequest, err := http.NewRequest(httpRequest.Method, address+httpRequest.RequestURI, strings.NewReader(string(body)))
			if err == nil {
				// httpClient := &http.Client{}
				log.Print("redirect to ", rebuildRequest.Host, rebuildRequest.URL)
				response, err := httpClient.Do(rebuildRequest)
				if err == nil {
					code := response.StatusCode
					content, err := ioutil.ReadAll(response.Body)
					response.Body.Close()
					if err == nil {
						resultQueue <- ParallelResponse{request.id, address, code, content, response}
						return
					} else {
						log.Print(err.Error())
					}
				} else {
					log.Print(err.Error())
				}
				resultQueue <- ParallelResponse{request.id, address, 500, []byte("{}"), response}
			} else {
				resultQueue <- ParallelResponse{request.id, address, 500, []byte("{}"), nil}
			}
		}(request.hosts[index], request.httpRequest)
	}

	timeoutNotifier := make(chan int)
	go func(timeout time.Duration) {
		time.Sleep(timeout)
		timeoutNotifier <- 1
	}(globalTimeout + time.Second*2)

	resultCount := 0
	keepWaiting := true
	allResponse := make([]string, 0)
	var chosenResponse *http.Response = nil
	for keepWaiting {
		select {
		case result := <-resultQueue:
			fmt.Println("get Response")
			resultCount++
			allResponse = append(allResponse, strings.Trim(string(result.content), "\n"))
			if chosenResponse == nil && result.code == 200 {
				chosenResponse = result.httpResponse
			}
		case <-timeoutNotifier:
			keepWaiting = false
		}
		if resultCount == len(request.hosts) {
			keepWaiting = false
		}
	}
	combinedResponse := ""
	if len(allResponse) > 1 {
		combinedResponse = strings.Join(allResponse, ",")
	} else if len(allResponse) == 1 {
		combinedResponse = allResponse[0]
	}
	combinedResponse = "{ response: [" + combinedResponse + "] }"
	if chosenResponse == nil {
		chosenResponse = &http.Response{}
	}
	return *chosenResponse, combinedResponse
}

func HelloServer(w http.ResponseWriter, req *http.Request) {
	log.Print("Receive: ", req.Host, req.URL)
	log.Print("Redirecting...")
	request := NewParallelRequest(req)
	response, content := Do(request, 10*time.Second)
	for k, v := range response.Header {
		for _, vv := range v {
			w.Header().Set(k, vv)
		}
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Write([]byte(content))
	log.Print("Done")
}

func main() {
	for i, v := range Hosts {
		fmt.Println(i, v)
	}
	http.HandleFunc("/hello", HelloServer)
	log.Print("server start at: 8612")
	err := http.ListenAndServe(":8612", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}
