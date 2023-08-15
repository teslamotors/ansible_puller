// Webserver logic

package main

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

const (
	httpPathAnsibleAdhocTrigger = "/ansible/adhoc-run"
	httpPathAnsibleDisable      = "/ansible/disable"
	httpPathAnsibleEnable       = "/ansible/enable"
	httpPathAnsibleControl      = "/ansible/control"
	httpPathStatus              = "/ansible/status"
)

var (
	disableReason string

	//go:embed templates/index.html
	indexHtml string

	//go:embed templates/ansible_controller.html
	ansibleController string
)

// MakeRunOnceHandler returns an http handler that calls runOnce when invoked.
func MakeRunOnceHandler(runOnce func()) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		runOnce()
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

	t, _ := template.New("foo").Parse(ansibleController)
	_ = t.Execute(w, data)
}

func HandlerIndex(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Hostname string
	}{
		hostname,
	}

	t, _ := template.New("foo").Parse(indexHtml)
	_ = t.Execute(w, data)
}

func HandlerStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"app_name":                 appName,
		"hostname":                 hostname,
		"ansible_disabled":         ansibleDisabled,
		"ansible_running":          ansibleRunning,
		"ansible_last_run_success": ansibleLastRunSuccess,
		"version":                  Version,
	}

	data, err := json.Marshal(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// NewServer creates a new http server
//
// runOnce is a function that we will called when the adhocTrigger handler is invoked.
func NewServer(runOnce func()) *http.Server {
	r := mux.NewRouter()

	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/", HandlerIndex).Methods("GET")
	r.HandleFunc(httpPathAnsibleAdhocTrigger, MakeRunOnceHandler(runOnce)).Methods("POST")
	r.HandleFunc(httpPathAnsibleDisable, HandlerAnsibleDisable).Methods("POST")
	r.HandleFunc(httpPathAnsibleEnable, HandlerAnsibleEnable).Methods("POST")
	r.HandleFunc(httpPathAnsibleControl, HandlerAnsibleControl).Methods("GET")
	r.HandleFunc(httpPathStatus, HandlerStatus).Methods("GET")

	srv := &http.Server{
		Handler:      r,
		Addr:         viper.GetString("http-listen-string"),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return srv
}
