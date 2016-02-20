package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/go-mecab"
)

var model mecab.Model

func main() {
	var err error
	model, err = mecab.NewModel(map[string]string{})
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	tagger, err := model.NewMeCab()
	if err != nil {
		panic(err)
	}

	query := r.URL.Query()
	result, err := tagger.Parse(query.Get("sentense"))
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, result)
}
