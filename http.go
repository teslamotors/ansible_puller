// Webserver logic

package main

import (
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"html/template"
	"net/http"
	"time"
)

const (
	httpPathAnsibleAdhocTrigger = "/ansible/adhoc-run"
	httpPathAnsibleDisable      = "/ansible/disable"
	httpPathAnsibleEnable       = "/ansible/enable"
	httpPathAnsibleControl      = "/ansible/control"
)

var (
	disableReason string
	htmlTemplates = packr.New("myHtml", "./templates")
)

// MakeChannelHandler returns an http handler that populates a channel with a single `true` when the handler is invoked
func MakeChannelHandler(c chan bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c <- true
		http.Redirect(w, r, httpPathAnsibleControl, http.StatusFound)
	}
}

func HandlerAnsibleEnable(w http.ResponseWriter, r *http.Request) {
	disableReason = ""

	ansibleEnable()
	http.Redirect(w, r, httpPathAnsibleControl, http.StatusFound)
}

func HandlerAnsibleDisable(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if val, ok := r.Form["disable-reason"]; ok {
		disableReason = val[0]
	}

	ansibleDisable()
	http.Redirect(w, r, httpPathAnsibleControl, http.StatusFound)
}

func HandlerAnsibleControl(w http.ResponseWriter, r *http.Request) {
	data := struct {
		AnsibleDisabled       bool
		AnsibleLastRunSuccess bool
		JobRunning            bool
		Hostname              string
		DisableReason         string
	}{
		ansibleDisabled, // Callout to the global var in main... inelegant
		ansibleLastRunSuccess,
		ansibleRunning,
		hostname,
		disableReason,
	}

	htmlStr, err := htmlTemplates.FindString("ansible_controller.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t, _ := template.New("foo").Parse(htmlStr)
	_ = t.Execute(w, data)
}

func HandlerIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Hostname string
	}{
		hostname,
	}

	htmlStr, err := htmlTemplates.FindString("index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t, _ := template.New("foo").Parse(htmlStr)
	_ = t.Execute(w, data)
}

// NewServer creates a new http server
//
// runChan is a channel that we will write to when the adhocTrigger handler is invoked.
func NewServer(runChan chan bool) *http.Server {
	r := mux.NewRouter()

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/", HandlerIndex).Methods("GET")
	r.HandleFunc(httpPathAnsibleAdhocTrigger, MakeChannelHandler(runChan)).Methods("POST")
	r.HandleFunc(httpPathAnsibleDisable, HandlerAnsibleDisable).Methods("POST")
	r.HandleFunc(httpPathAnsibleEnable, HandlerAnsibleEnable).Methods("POST")
	r.HandleFunc(httpPathAnsibleControl, HandlerAnsibleControl).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         viper.GetString("http-listen-string"),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return srv
}
