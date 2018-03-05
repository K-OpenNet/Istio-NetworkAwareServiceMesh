package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var err error

type JSONCmd struct {
	CmdType    string
	CmdForm    string
	CmdContent json.RawMessage
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}

		reqBody, _ := ioutil.ReadAll(r.Body)
		reqStr := string(reqBody)
		var cmd JSONCmd
		err = json.Unmarshal([]byte(reqStr), &cmd)

		if err != nil {
			http.Error(w, err.Error(), 400)
			fmt.Println("Error occured")
			return
		}
		fmt.Println("ReqBody: " + reqStr)
		fmt.Println("CmdType: " + cmd.CmdType)
		fmt.Println("CmdContent: " + string(cmd.CmdContent))
		fmt.Println("CmdForm: " + cmd.CmdForm)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
