package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lestrrat/go-server-starter/listener"
	"github.com/shogo82148/go-gracedown"
	"github.com/shogo82148/go-mecab"
)

var model mecab.Model

func main() {
	signal_chan := make(chan os.Signal)
	signal.Notify(signal_chan, syscall.SIGTERM)
	go func() {
		for {
			s := <-signal_chan
			if s == syscall.SIGTERM {
				gracedown.Close()
			}
		}
	}()

	listeners, err := listener.ListenAll()
	if err != nil && err != listener.ErrNoListeningTarget {
		panic(err)
	}
	var l net.Listener
	if err == listener.ErrNoListeningTarget {
		// Fallback if not running under Server::Starter
		l, err = net.Listen("tcp", ":8080")
		if err != nil {
			panic("Failed to listen to port 8080")
		}
	} else {
		l = listeners[0]
	}

	model, err = mecab.NewModel(map[string]string{})
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	gracedown.Serve(l, mux)
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
