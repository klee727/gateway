package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
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
	err          error
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

/**
* @brief 重设Request,
*
* @param http.Request
* @param
* @param error
*
* @return
 */
func ResetRequest(request *http.Request, hostAddress string) (*http.Request, error) {
	newUrl, err := url.Parse(hostAddress + request.RequestURI)
	// 清除RequestURI
	request.RequestURI = ""
	// 重新设置地址
	request.URL = newUrl
	return request, err
}

/**
* @brief 发送请求
*
* @param ParallelRequest
* @param
* @param string
*
* @return
 */

func Do(request *ParallelRequest, globalTimeout time.Duration) (*http.Response, string) {
	resultQueue := make(chan ParallelResponse)
	for index, _ := range request.hosts {
		go func(hostAddress string, httpRequest http.Request) {
			httpClient := &http.Client{
				//	Transport: &http.Transport{
				//		Dial: func(netw, addr string) (net.Conn, error) {
				//			c, err := net.DialTimeout(netw, addr, globalTimeout)
				//			if err != nil {
				//				return nil, err
				//			}
				//			deadline := time.Now().Add(2 * globalTimeout)
				//			c.SetDeadline(deadline)
				//			return c, nil
				//		},
				//		ResponseHeaderTimeout: globalTimeout,
				//	},
				Timeout: globalTimeout,
			}
			// 转发 Request
			rebuiltRequest, err := ResetRequest(&httpRequest, hostAddress)
			if err != nil {
				log.Println("fail to rebuild request: ", err.Error())
				resultQueue <- ParallelResponse{request.id, hostAddress, err, nil}
				return
			}
			// httpClient := &http.Client{}
			log.Print("redirect to ", rebuiltRequest.Host, rebuiltRequest.URL)
			response, err := httpClient.Do(rebuiltRequest)
			if err != nil {
				log.Print("fail to send request: ", err.Error())
				resultQueue <- ParallelResponse{request.id, hostAddress, err, response}
				return
			}
			resultQueue <- ParallelResponse{request.id, hostAddress, err, response}
		}(request.hosts[index], request.httpRequest)
	}

	timeoutNotifier := make(chan int)
	go func(timeout time.Duration) {
		time.Sleep(timeout)
		timeoutNotifier <- 1
	}(globalTimeout + time.Second*2)

	resultCount := 0
	keepWaiting := true
	responseCollection := make([]string, 0)
	var chosenResponse *http.Response = nil
	for keepWaiting {
		select {
		case result := <-resultQueue:
			fmt.Println("get Response from ", result.id)
			content := "{}"
			if result.err != nil {
				// TODO. 造一个统一的错误数据结构
				content = fmt.Sprintf("{\"error\":\"%v\"}", result.err.Error())
			} else if body, err := ioutil.ReadAll(result.httpResponse.Body); err == nil {
				content = strings.Trim(string(body), "\n")
				// 随便选一个Response作为最终的header
				if chosenResponse == nil {
					chosenResponse = result.httpResponse
				}
			}
			responseCollection = append(responseCollection, content)
			resultCount++
		case <-timeoutNotifier:
			keepWaiting = false
		}
		if resultCount == len(request.hosts) {
			keepWaiting = false
		}
	}
	combinedResponse := strings.Join(responseCollection, ",")
	combinedResponse = "{ response: [" + combinedResponse + "] }"
	return chosenResponse, combinedResponse
}

type Router struct {
	routerMap map[string]func(w http.ResponseWriter, req *http.Request)
}

func (router *Router) InitRouterTable() {

}

func Redirect(w http.ResponseWriter, req *http.Request) {
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

func (router *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	Redirect(w, req)
}

/**
* @brief 命令行参数
*
* @return
 */

type ServerConfigure struct {
	host         string
	port         string
	readTimeout  int
	writeTimeout int
}

var serverConfigure *ServerConfigure = &ServerConfigure{}

func Init() {
	flag.StringVar(&serverConfigure.host, "host", "0.0.0.0", "host for server to listen on")
	flag.StringVar(&serverConfigure.port, "port", "8612", "port for server to listen on")
	flag.IntVar(&serverConfigure.readTimeout, "read_timeout", 30, "maximum duration before timing out read of the request (second)")
	flag.IntVar(&serverConfigure.writeTimeout, "write_timeout", 30, "maximum duration before timing out write of the request (second)")
	flag.Parse()
}

func main() {
	Init()
	for i, v := range Hosts {
		fmt.Println(i, v)
	}
	httpServer := http.Server{
		Addr:         serverConfigure.host + ":" + serverConfigure.port,
		Handler:      &Router{},
		ReadTimeout:  time.Duration(serverConfigure.readTimeout) * time.Second,
		WriteTimeout: time.Duration(serverConfigure.writeTimeout) * time.Second,
	}
	log.Print("server ready to accept request, configure:")
	configureValue := reflect.ValueOf(*serverConfigure)
	configureType := reflect.TypeOf(*serverConfigure)
	for i := 0; i < configureValue.NumField(); i++ {
		log.Printf("\t%v:\t%v\n", configureType.Field(i).Name, configureValue.Field(i))
	}
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal("server down: ", err)
	}
}
