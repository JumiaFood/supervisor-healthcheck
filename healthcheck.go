package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"log"
	"net/url"
	"github.com/kolo/xmlrpc"
	"os"
	"path"
)

type HealthCheck struct {
	Status          bool `json:"status"`
	SupervisorState *SupervisorState `json:"supervisor_state,omitempty"`
	Messages        []string `json:"messages,omitempty"`
}

type SupervisorState struct {
	StateCode int `xmlrpc:"statecode" json:"state_code"`
	StateName string `xmlrpc:"statename" json:"state_name"`
}

type SupervisorProcessInfo struct {
	Name          string `xmlrpc:"name"`
	Group         string `xmlrpc:"group"`
	Description   string `xmlrpc:"description"`
	Start         int `xmlrpc:"start"`
	Stop          int `xmlrpc:"stop"`
	Now           int `xmlrpc:"now"`
	State         int `xmlrpc:"state"`
	StateName     string `xmlrpc:"statename"`
	SpawnErr      string `xmlrpc:"spawnerr"`
	ExitStatus    int `xmlrpc:"exitstatus"`
	LogFile       string `xmlrpc:"logfile"`
	StdoutLogfile string `xmlrpc:"stdout_logfile"`
	StderrLogfile string `xmlrpc:"stderr_logfile"`
	Pid           int `xmlrpc:"pid"`
}

func SupervisorUrl() *url.URL {
	endpoint, err := url.ParseRequestURI(fmt.Sprintf("http://%s:%s", os.Getenv("SUPERVISOR_HOST"), os.Getenv("SUPERVISOR_PORT")))
	if err != nil {
		log.Fatal(err)
	}
	return endpoint
}

func SupervisorRpcEndpoint() string {
	u := SupervisorUrl()
	u.Path = path.Join(u.Path, "/RPC2")
	return u.String()
}

func MarshalHealthCheck(health HealthCheck) string {
	bytes, err := json.Marshal(health)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func main() {
	client, err := xmlrpc.NewClient(SupervisorRpcEndpoint(), nil)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})

	http.HandleFunc("/health/check", func(w http.ResponseWriter, r *http.Request) {
		var health HealthCheck

		var state SupervisorState
		err = client.Call("supervisor.getState", nil, &state)
		if err != nil {
			health.Status = false
			health.Messages = append(health.Messages, err.Error())
		} else {
			health.Status = (state.StateCode == 1)
			health.SupervisorState = &state
		}

		var processes []SupervisorProcessInfo
		err = client.Call("supervisor.getAllProcessInfo", nil, &processes)
		if err != nil {
			health.Status = false
			health.Messages = append(health.Messages, err.Error())
		} else {
			var failed []string
			for _, v := range processes {
				if v.State == 200 || v.StateName == "FATAL" {
					failed = append(failed, v.Name)
				}
			}
			health.Status = (health.Status && len(failed) == 0)
			health.Messages = append(health.Messages, failed...)
		}

		if (health.Status == false) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		fmt.Fprintf(w, MarshalHealthCheck(health))
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
