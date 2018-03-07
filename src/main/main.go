package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"bytes"
	"strings"
	"regexp"
)

var err error

var host = "210.117.251.17"
var user = "mjkang"
var ssh_key = "~/.ssh/netcs.key.plain"

var validPath = regexp.MustCompile("^/(generic|k8s|istio)$")

type JSONCmd struct {
	CmdType    string
	CmdForm    string
	CmdEmbededJSON []json.RawMessage
}

func main() {
	http.HandleFunc("/generic", makeHandler(genericHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func genericHandler(w http.ResponseWriter, r *http.Request) {

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
		
		// Stripping unnecessary whitespaces
		var b strings.Builder
		for _, obj := range cmd.CmdEmbededJSON {
			jsonByte, _ := json.Marshal(obj)
			fmt.Fprintf(&b, string(jsonByte))
			fmt.Fprintf(&b, " ")
		}
		s := b.String()
		s = s[:len(s)-1]
				
		fmt.Println("ReqBody: " + reqStr)
		fmt.Println("CmdType: " + cmd.CmdType)
		fmt.Println("CmdEmbededJSON: " + s)
		fmt.Println("CmdForm: " + cmd.CmdForm)
		
		cmdFull := strings.Replace(cmd.CmdForm, "$EmbededJSON", s, -1)
		fmt.Println("CmdFull: " + cmdFull)
		
		cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmdFull)
		var out bytes.Buffer
		var errout bytes.Buffer
		cmdObj.Stdout = &out
		cmdObj.Stderr = &errout
		err = cmdObj.Run()
		if err != nil {
			fmt.Println(errout.String())
			fmt.Fprintf(w, errout.String(), r.URL.Path[1:])
			// log.Fatal(err)
		}
		fmt.Println(out.String())
		fmt.Fprintf(w, out.String(), r.URL.Path[1:])
	}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        
    		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}

        fn(w, r)
    }
}

func stripChars(str, chr string) string {
    return strings.Map(func(r rune) rune {
        if strings.IndexRune(chr, r) < 0 {
            return r
        }
        return -1
    }, str)
}
