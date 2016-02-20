package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lestrrat/go-server-starter/listener"
	"github.com/shogo82148/go-gracedown"
	"github.com/shogo82148/go-mecab"
)

type APIResponse struct {
	MeCabIPADIC []Node `json:"mecab_ipadic"`
}

type Node struct {
	Surface  string `json:"surface"`
	Feature  string `json:"feature"`
	POS      string `json:"pos"`
	Baseform string `json:"baseform,omitempty"`
	Reading  string `json:"reading,omitempty"`
}

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
	query := r.URL.Query()
	sentense := query.Get("sentense")
	result := APIResponse{
		MeCabIPADIC: parseMeCabIPADIC(sentense),
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}

func parseMeCabIPADIC(sentense string) []Node {
	tagger, err := model.NewMeCab()
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	nodes := []Node{}
	node, err := tagger.ParseToNode(sentense)
	if err != nil {
		panic(err)
	}

	for ; node != (mecab.Node{}); node = node.Next() {
		if stat := node.Stat(); stat == mecab.BOSNode || stat == mecab.EOSNode {
			continue
		}
		feature := node.Feature()
		features := strings.Split(feature, ",")
		posElem := make([]string, 0, 3)
		for _, e := range features[:4] {
			if e != "*" {
				posElem = append(posElem, e)
			}
		}
		reading := ""
		if len(features) > 7 {
			reading = features[7]
		}
		baseform := ""
		if len(features) > 6 {
			baseform = features[6]
		}
		nodes = append(nodes, Node{
			Surface:  node.Surface(),
			Feature:  feature,
			POS:      strings.Join(posElem, "-"),
			Reading:  reading,
			Baseform: baseform,
		})
	}

	return nodes
}
