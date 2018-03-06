package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"bytes"
)

var err error

var host = "210.117.251.17"
var user = "mjkang"
var ssh_key = "~/.ssh/netcs.key.plain"

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

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error occured while reading the request body", 400)
			return
		}

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
		
		if (cmd.CmdType == "k8s" && cmd.CmdForm == "apply") {
//			bash -c "echo blahblah | cat"
//			cmd := exec.Command("bash", "-c", "echo blahblah | cat")
//			var head = ""
//			cmd := "kubectl apply"	
//			cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmd)
			cmd := "cat > blah.txt << EOF\n" + string(cmd.CmdContent) + "\n"
			cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmd)
			var out bytes.Buffer
			cmdObj.Stdout = &out
			err := cmdObj.Run()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(out.String())			
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
