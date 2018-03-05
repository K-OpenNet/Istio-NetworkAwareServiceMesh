package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var err error

type User struct {
	Id      string
	Balance uint64
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// var u User
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		s := string(body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// fmt.Println(u.Id)
		fmt.Println(s)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
