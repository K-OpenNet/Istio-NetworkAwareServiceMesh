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

// Required constants for remote host
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

type K8sGetOpt struct {
	Namespace string
	Selectors []Selector
}

type Selector struct {
	Operator rune
	Label string
	Value string
}

func main() {
	http.HandleFunc("/generic", makeHandler(genericHandler))
	http.HandleFunc("/k8s/apply", makeHandler(k8sApplyHandler))
	http.HandleFunc("/k8s/replace", makeHandler(k8sReplaceHandler))
	http.HandleFunc("/k8s/delete", makeHandler(k8sDeleteHandler))
	http.HandleFunc("/k8s/get", makeHandler(k8sGetHandler))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func k8sGetHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var embedJSON K8sGetOpt
		err = json.Unmarshal(cmd.CmdEmbededJSON[0], &embedJSON)
		if err != nil {
			http.Error(w, "Error occured during unmarshalling the embeded JSON", 400)
			return
		}	
	
		var b strings.Builder
		fmt.Fprintf(&b, "kubectl get " + cmd.CmdForm + " -o json")		
		fmt.Fprintf(&b, " -n " + embedJSON.Namespace)
		// Selector option expected output: -l key1=value1,key2=value2
		if (embedJSON.Selectors != nil && len(embedJSON.Selectors) > 0) {
			fmt.Fprintf(&b, " -l ")
			for pos, sel := range embedJSON.Selectors {
				fmt.Fprintf(&b, sel.Label)
				fmt.Fprintf(&b, string(sel.Operator))
				fmt.Fprintf(&b, sel.Value)
				if (pos < len(embedJSON.Selectors) - 1) {
					fmt.Fprintf(&b, ",")	
				}
			}
		}
		cmdFull := b.String()
		fmt.Println("CmdFull: " + cmdFull)
		
		renderJSONResponse(w, r, cmdFull)
	}

func k8sDeleteHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var b strings.Builder
		fmt.Fprintf(&b, "cat <<EOF | kubectl delete -o name -f -\n")
		fmt.Fprintf(&b, rBodyStr)
		fmt.Fprintf(&b, "\n")
		cmdFull := b.String()
		
		fmt.Println("CmdFull: " + cmdFull)
		
		renderPlainResponse(w, r, cmdFull)
	}

func k8sApplyHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var b strings.Builder
		fmt.Fprintf(&b, "cat <<EOF | kubectl apply -o json -f -\n")
		fmt.Fprintf(&b, rBodyStr)
		fmt.Fprintf(&b, "\n")
		cmdFull := b.String()
		
		fmt.Println("CmdFull: " + cmdFull)
		
		renderJSONResponse(w, r, cmdFull)
	}

func k8sReplaceHandler(w http.ResponseWriter, r *http.Request, cmd JSONCmd, rBodyStr string) {
		var b strings.Builder
		fmt.Fprintf(&b, "cat <<EOF | kubectl replace -o json -f -\n")
		fmt.Fprintf(&b, rBodyStr)
		fmt.Fprintf(&b, "\n")
		cmdFull := b.String()
		
		fmt.Println("CmdFull: " + cmdFull)
		
		renderJSONResponse(w, r, cmdFull)
	}

func renderPlainResponse(w http.ResponseWriter, r *http.Request, cmdFull string) {
		out, outerr, err := execCmd(cmdFull)
		if err != nil {
			fmt.Println(outerr)
			fmt.Fprintf(w, outerr, r.URL.Path[1:])
			// log.Fatal(err)
		}
		
		fmt.Println(out)
		fmt.Fprintf(w, out, r.URL.Path[1:])	
}

func renderJSONResponse(w http.ResponseWriter, r *http.Request, cmdFull string) {
		out, outerr, err := execCmd(cmdFull)
		if err != nil {
			http.Error(w, outerr, 400)
			return
		}

		// This is required because kubectl JSON response is not given in an array.
		// The response is given like multiple JSON with one large object, and
		// the JSONs are separated by only new line char. 
		depth := 0
		bEsc := false
		pHead := 0
		jRes := make([]json.RawMessage, 0, 2)
		for pos, char := range out {
			if (char == '\x22') { // '"' double quote char
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
					pStr := out[pHead:pos + 1]
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
		
		renderPlainResponse(w, r, cmdFull)
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

func execCmd(cmdFull string) (string, string, error) {
	cmdObj := exec.Command("ssh", "-i", ssh_key, user + "@" + host, cmdFull)
	var out, outerr bytes.Buffer
	cmdObj.Stdout = &out
	cmdObj.Stderr = &outerr
	err := cmdObj.Run()
	
	return out.String(), outerr.String(), err
}

func stripChars(str, chr string) string {
    return strings.Map(func(r rune) rune {
        if strings.IndexRune(chr, r) < 0 {
            return r
        }
        return -1
    }, str)
}
