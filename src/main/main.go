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

/*
	Possible pathes
	/generic
	/k8s/apply
	/k8s/delete
	/k8s/get
 */
var validPath = regexp.MustCompile("((^/generic$)|(^/k8s/(apply|delete|get)$)|(^/istio/(apply|delete|get)$))")

type JSONCmd struct {
	CmdType    string
	CmdForm    string
	CmdEmbededJSON []json.RawMessage
}

func main() {
	http.HandleFunc("/generic", makeHandler(genericHandler))
	http.HandleFunc("/k8s/apply", makeHandler(k8sApplyHandler))
	http.HandleFunc("/k8s/delete", makeHandler(k8sDeleteHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func k8sDeleteHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var b strings.Builder
		fmt.Fprintf(&b, "cat <<EOF | kubectl delete -o name -f -\n")
		fmt.Fprintf(&b, rBodyStr)
		fmt.Fprintf(&b, "\n")
		cmdFull := b.String()
		
		fmt.Println("CmdFull: " + cmdFull)
		
		cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmdFull)
		renderPlainResponse(w, r, cmdObj)
	}

func k8sApplyHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var b strings.Builder
		fmt.Fprintf(&b, "cat <<EOF | kubectl apply -o json -f -\n")
		fmt.Fprintf(&b, rBodyStr)
		fmt.Fprintf(&b, "\n")
		cmdFull := b.String()
		
		fmt.Println("CmdFull: " + cmdFull)
		
		cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmdFull)
		renderJSONResponse(w, r, cmdObj)
	}

func renderPlainResponse(w http.ResponseWriter, r *http.Request, cmdObj *exec.Cmd) {
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

func renderJSONResponse(w http.ResponseWriter, r *http.Request, cmdObj *exec.Cmd) {
		var out bytes.Buffer
		var errout bytes.Buffer
		cmdObj.Stdout = &out
		cmdObj.Stderr = &errout
		err = cmdObj.Run()
		if err != nil {
			http.Error(w, errout.String(), 400)
			return
		}

		s := out.String()
		depth := 0
		bEsc := false
		pHead := 0
		jRes := make([]json.RawMessage, 0, 2)
		for pos, char := range s {
			if (char == '\x22') {
				if bEsc {
					bEsc = false
				} else {
					bEsc = true
				}
				
				continue
			}
			if (char == '{') {
				depth++
				if (depth == 1) {
					pHead = pos
				}
				continue			
			}
			if (char == '}') {
				depth--
				if (depth == 0) {
					pStr := s[pHead:pos + 1]
					fmt.Println("pStr : " + pStr)
					res, _ := json.Marshal(json.RawMessage(pStr))
					fmt.Println("Marshalled : " + string(res))
					jRes = append(jRes, res)				
				}
				continue			
			}			
		}
		res, _ := json.Marshal(jRes)
		fmt.Println(string(res))
		
		json.NewEncoder(w).Encode(jRes)
}

func genericHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		cmdFull := strings.Replace(cmd.CmdForm, "$EmbededJSON", rBodyStr, -1)
		fmt.Println("CmdFull: " + cmdFull)
		
		cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmdFull)
		renderPlainResponse(w, r, cmdObj)
	}

func makeHandler(fn func(http.ResponseWriter, *http.Request, JSONCmd, string)) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
	    	// Validating given URL path
        m := validPath.FindStringSubmatch(r.URL.Path)
        if m == nil {
            http.NotFound(w, r)
            return
        }
        
    		// Checking if the body is OK
    		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		rBodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error occured while reading the request body", 400)
			return
		}

		var cmd JSONCmd
		err = json.Unmarshal(rBodyBytes, &cmd)

		if err != nil {
			http.Error(w, "Error occured during unmarshalling the request body", 400)
			return
		}
		
		// Stripping unnecessary whitespaces
		var b strings.Builder
		for _, obj := range cmd.CmdEmbededJSON {
			obj, _ := json.Marshal(obj)
			fmt.Fprintf(&b, string(obj))
			fmt.Fprintf(&b, " ")
		}
		rBodyStr := b.String()
		rBodyStr = rBodyStr[:len(rBodyStr)-1]
				
		fmt.Println("RemoteAddr: " + r.RemoteAddr)
		fmt.Println("CmdType: " + cmd.CmdType)
		fmt.Println("CmdForm: " + cmd.CmdForm)
		fmt.Println("CmdEmbededJSON: " + rBodyStr)
        fn(w, r, cmd, rBodyStr)
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
