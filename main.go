package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"gopkg.in/yaml.v2"

	"github.com/lestrrat/go-server-starter/listener"
	"github.com/shogo82148/go-gracedown"
	"github.com/shogo82148/go-mecab"
)

type APIResponse struct {
	MeCabIPADIC    []Node `json:"mecab_ipadic,omitempty"`
	MeCabNEologd   []Node `json:"mecab_neologd,omitempty"`
	NEologdVersion string `json:"neologd_version,omitempty"`
	MeCabUnidic    []Node `json:"mecab_unidic,omitempty"`
}

type Node struct {
	Surface  string `json:"surface"`
	Feature  string `json:"feature"`
	POS      string `json:"pos"`
	Baseform string `json:"baseform,omitempty"`
	Reading  string `json:"reading,omitempty"`
}

type NEologdConfig struct {
	Dicdir  string `yaml:"dicdir"`
	Version string `yaml:"version"`
}

var mecabdic string
var modelIPADIC mecab.Model
var modelNEologd mecab.Model
var modelUnidic mecab.Model
var neologdConfig NEologdConfig
var unidicAvailable bool

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

	b, err := exec.Command("mecab-config", "--dicdir").Output()
	if err != nil {
		panic(err)
	}
	mecabdic = strings.TrimSpace(string(b))

	// load dictionaries
	modelIPADIC, err = mecab.NewModel(map[string]string{})
	if err != nil {
		panic(err)
	}

	buf, err := ioutil.ReadFile("neologd-config.yml")
	if err == nil {
		yaml.Unmarshal(buf, &neologdConfig)
		modelNEologd, err = mecab.NewModel(map[string]string{"dicdir": neologdConfig.Dicdir})
		if err != nil {
			panic(err)
		}
	}

	modelUnidic, err = mecab.NewModel(map[string]string{"dicdir": mecabdic + "/unidic"})
	unidicAvailable = err == nil

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	gracedown.Serve(l, mux)
}

func handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	r.ParseMultipartForm(1024)
	sentense := r.Form.Get("sentense")
	parsers := r.Form.Get("parsers")
	if parsers == "" {
		parsers = "mecab_ipadic"
	}
	parsersMap := map[string]struct{}{}
	for _, parser := range strings.Split(parsers, ",") {
		parsersMap[parser] = struct{}{}
	}

	result := APIResponse{}
	if _, ok := parsersMap["mecab_ipadic"]; ok {
		result.MeCabIPADIC = parseMeCabIPADIC(sentense)
	}
	if _, ok := parsersMap["mecab_neologd"]; ok && neologdConfig.Dicdir != "" {
		result.MeCabNEologd = parseMeCabNEologd(sentense)
		result.NEologdVersion = neologdConfig.Version
	}
	if _, ok := parsersMap["mecab_unidic"]; ok && unidicAvailable {
		result.MeCabUnidic = parseMeCabUnidic(sentense)
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(result)
}

func parseMeCabIPADIC(sentense string) []Node {
	tagger, err := modelIPADIC.NewMeCab()
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	node, err := tagger.ParseToNode(sentense)
	if err != nil {
		panic(err)
	}
	return node2struct(node)
}

func parseMeCabNEologd(sentense string) []Node {
	tagger, err := modelNEologd.NewMeCab()
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	node, err := tagger.ParseToNode(sentense)
	if err != nil {
		panic(err)
	}
	return node2struct(node)
}

func parseMeCabUnidic(sentense string) []Node {
	tagger, err := modelUnidic.NewMeCab()
	if err != nil {
		panic(err)
	}
	defer tagger.Destroy()

	node, err := tagger.ParseToNode(sentense)
	if err != nil {
		panic(err)
	}

	nodes := []Node{}
	for ; node != (mecab.Node{}); node = node.Next() {
		if stat := node.Stat(); stat == mecab.BOSNode || stat == mecab.EOSNode {
			continue
		}
		feature := node.Feature()
		features, _ := splitFeature(feature)
		posElem := make([]string, 0, 3)
		for _, e := range features[:4] {
			if e != "*" {
				posElem = append(posElem, e)
			}
		}
		reading := ""
		if len(features) > 6 {
			reading = features[6]
		}
		baseform := ""
		if len(features) > 8 {
			baseform = features[8]
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

func node2struct(node mecab.Node) []Node {
	nodes := []Node{}
	for ; node != (mecab.Node{}); node = node.Next() {
		if stat := node.Stat(); stat == mecab.BOSNode || stat == mecab.EOSNode {
			continue
		}
		feature := node.Feature()
		features, _ := splitFeature(feature)
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

func splitFeature(feature string) ([]string, error) {
	reader := bytes.NewBufferString(feature)
	return csv.NewReader(reader).Read()
}
