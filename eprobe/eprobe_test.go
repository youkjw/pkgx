package eprobe

import (
	"net/http"
	"testing"
	"time"
)

type probe struct {
}

func (p *probe) LivenessProbe() EprobeState {
	return EprobeStateOK
}

func (p *probe) ReadinessProbe() EprobeState {
	return EprobeStateOK
}

func (p *probe) StartupProbe() EprobeState {
	return EprobeStateOK
}

func TestEprobeHandler(t *testing.T) {
	serve := &http.Server{
		Addr:         ":8082",
		Handler:      http.NewServeMux(),
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
	go serve.ListenAndServe()
}
