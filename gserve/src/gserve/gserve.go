package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"html/template"	
)

var zookeeper string = "zookeeper"
var hbase_host string = "hbase"
var server_name string = "Unknown server"
const tpl = `
<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>SE2 Assignment 1: Library</title>
</head>


    <body>
        <div class="container">
            <h1>SE2 Library</h1>
             {{range .Row}}
                <div class="row card-panel">
                    <h2 class="pink-text">{{.Key}}</h2>
                    <div class="document">
                        <div class="s12 row" style="margin-left:auto;">
                            <h4 class="s12 flow-text"> Document</h4>
                            {{range .Data}} {{ $prefix := stringSlice .Column 0 4 }} {{if (eq $prefix "docu") }}
                            <div class="col s5">
                                {{ stringSlice .Column 9 -1}}
                            </div>
                            <div class="col s7">
                                {{stringSlice .Text 6 -1}}
                            </div>
                            {{end}} {{end}}
                        </div> <br>
                        <div class="s12 ">
                            <h4 class="s12 flow-text"> Metadata</h4>
                            {{range .Data}}
                                {{ $prefix := stringSlice .Column 0 4 }}                          
                                {{if (eq $prefix "meta") }} 
                                    <div class="col s5">
                                        {{stringSlice .Column 9 -1}}
                                    </div>
                                    <div class="col s7">
                                        {{stringSlice .Text 6 -1}}
                                    </div>
                                {{end}}
                            {{end}}
                        </div>

                    </div>                       


                </div>
             {{end}}
        </div>
    </body>
</html>`

// The data structure in accordance with response of Hbase
type HbaseResp struct {
    Row []Document `json:"Row"`
}
type Document struct {
    Key string `json:"key"`
    Data []Cell `json:"Cell"`
}
type Cell struct {
    Column string `json:"column"`
	Text string `json:"$"`
	Time int64 `json:"timestamp"`	
}

// Functions
func must(err error) {
	if err != nil {
		//panic(err)
		fmt.Printf("%+v From must \n", err)
	}
}
func connect() *zk.Conn {
	zksStr := zookeeper + ":2181"
	zks := strings.Split(zksStr, ",")
	conn, _, err := zk.Connect(zks, time.Second)
	must(err)
	return conn
}


func encoder(unencodedJSON []byte) string {
	
	var unencodedRows RowsType
	json.Unmarshal(unencodedJSON, &unencodedRows)

	encodedRows := unencodedRows.encode()

	encodedJSON, _ := json.Marshal(encodedRows)

	return string(encodedJSON)
}
func decoder(encodedJSON []byte) string {

	var encodedRows EncRowsType
	fmt.Println("\n Decoder input: ", string(encodedJSON))
	json.Unmarshal(encodedJSON, &encodedRows)
	fmt.Println("\n [1] obj from input: ", encodedRows)

	decodedRows, err := encodedRows.decode()
	must(err)
	fmt.Println("\n [2] rows from obj: ", decodedRows)
	// convert to json byte[] from go object (RowsType)
	deCodedJSON, _ := json.Marshal(decodedRows)

	return string(deCodedJSON)
}

// Hbase
func postToHbase(encodedJSON string) {

	req_url := "http://" + hbase_host + ":8080/se2:library/fakerow"

	resp, err := http.Post(req_url, "application/json", bytes.NewBuffer([]byte(encodedJSON)))

	if err != nil {
		fmt.Println("error from response: %+v", err)
		return
	}

	fmt.Println("Post Response: ", resp.Status)
	defer resp.Body.Close()
}
func getFromHbase() *HbaseResp {

	req_url := "http://" + hbase_host + ":8080/se2:library/*"

	req, _ := http.NewRequest("GET", req_url, nil)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{}
	resp, getErr := client.Do(req)
	must(getErr)

	fmt.Println("get response status: ", resp.Status)

	encodedJsonByte, err := ioutil.ReadAll(resp.Body)
	must(err)
	
	var respStr = new(HbaseResp)
	decodedJSON := decoder(encodedJsonByte)
	err = json.Unmarshal([]byte(decodedJSON), &respStr)
	must(err)
	defer resp.Body.Close()

	return respStr
}
func handler(writer http.ResponseWriter, req *http.Request) {

	if req.Method == "POST" || req.Method == "PUT" {

		encodedJsonByte, err := ioutil.ReadAll(req.Body)
		must(err)

		// get encoded data from []byte type
		encodedJSON := encoder(encodedJsonByte)
		fmt.Println("\n encoded JSON : ", string(encodedJSON))

		req.Header.Set("Content-type", "application/json")
		postToHbase(encodedJSON)
		fmt.Fprintf(writer, "an %s\n", "POST")

	} else if req.Method == "GET" {
		fmt.Printf("٩( 'ω' )و get！\n")

		req.Header.Set("Accept", "application/json")
		resData := getFromHbase()
		// Html response template
		//:= template.New("webpage").Parse(tpl)
    //		
		t := template.Must(template.New("dataTemplate").Funcs(template.FuncMap{
    		"stringSlice": func(s string, i, j int) string {
			awr := s
			if j==-1 {
			awr = s[i:]
			} else { 
			awr = s[i:j]
			} 
			return awr
			},
		}).Parse(tpl))
		//must(err)
		t.Execute(writer, resData)
	} else {
		fmt.Fprintf(writer, "Invalid request")
	}

	fmt.Fprintf(writer, "\n %s", server_name)

}
func startServer() {
	http.HandleFunc("/library", handler)
	log.Fatal(http.ListenAndServe(":9091", nil))
}

// START 
func main() {

	server_name = os.Getenv("servername")
	conn := connect()
	defer conn.Close()

	for conn.State() != zk.StateHasSession {
		fmt.Printf(" %s---?--->zookeeper ...\n", server_name)
		time.Sleep(time.Second * 4)
	}

    fmt.Printf(" %s is connected with Zookeeper\n", server_name)
	flags := int32(zk.FlagEphemeral)
	acl := zk.WorldACL(zk.PermAll)

	gserv, err := conn.Create("/grproxy/"+server_name, []byte(server_name+":9091"), flags, acl)
	must(err)
	fmt.Printf("create ephemeral node: %+v\n", gserv)

	startServer()
}
