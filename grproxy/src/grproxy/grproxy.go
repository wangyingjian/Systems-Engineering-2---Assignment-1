package main

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

var urls []string

func must(err error) {
	if err != nil {
		fmt.Printf("proxy :: %+v \n", err)
	}
}

// Create connecting to zookeeper
func connect() *zk.Conn {
	zksStr := "zookeeper:2181"
	zks := strings.Split(zksStr, ",")
	conn, _, err := zk.Connect(zks, time.Second)
	must(err)
	return conn
}

func monitorGserver(conn *zk.Conn, path string) (chan []string, chan error) {
	servers := make(chan []string)
	errors := make(chan error)
	go func() {
		for {
			children, _, events, err := conn.ChildrenW(path)
			if err != nil {
				errors <- err
				return
			}
			servers <- children
			evt := <-events
			if evt.Err != nil {
				errors <- evt.Err
				return
			}
		}
	}()
	return servers, errors
}

func MultipleReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		if req.URL.Path == "/library" {
			fmt.Println("?===>server")
			req.URL.Host = urls[rand.Int()%len(urls)]
			req.URL.Scheme = "http"
		} else {
			fmt.Println("?===>nginx")			
			req.URL.Host = "nginx"
			req.URL.Scheme = "http"
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func main() {
	conn := connect()
	defer conn.Close()

	flags := int32(0)
	acl := zk.WorldACL(zk.PermAll) // Access Control List

	// Waiting for Zookeeper
	// StateConnecting -> StateConnected -> StateHasSession
	for conn.State() != zk.StateHasSession {
		fmt.Printf("grproxy---?--->Zookeeper ...\n")
		time.Sleep(time.Second * 4)
	}

	// check/create proxy
	exists, stat, err := conn.Exists("/grproxy")
	must(err)
	if !exists {
		fmt.Printf("creating proxy...")
		grproxy, err := conn.Create("/grproxy", []byte("grproxy:80"), flags, acl)
		must(err)
		fmt.Printf("create: %+v\n", grproxy)
	} else {
		fmt.Printf("proxy exists: %+v %+v\n", exists, stat)
	}

	// Monitoring
	serverchn, errors := monitorGserver(conn, "/grproxy")

	go func() {
		for {
			select {
			case item := <-serverchn:
				fmt.Printf(" Server item: %+v \n", item)
				var temp []string
				for _, child := range item {
					gserve_urls, _, err := conn.Get("/grproxy/" + child)
					temp = append(temp, string(gserve_urls))
					if err != nil {
						fmt.Printf("error from child: %+v\n", err)
					}
				}
				urls = temp
				fmt.Printf(" Online urls: %+v \n", urls)
			case err := <-errors:
				fmt.Printf("monitoring gserver error: %+v \n", err)
			}
		}
	}()

	proxy := MultipleReverseProxy()
	log.Fatal(http.ListenAndServe(":8080", proxy))
}
