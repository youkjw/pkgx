package eprobe

import (
	"gitlab.cpp32.com/backend/epkg/utils/signal"
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

func TestEprobe(t *testing.T) {
	signald := signal.NewSignalHandle(nil)
	New(signald.GetContext()).Detect(&probe{})
	signald.WaitExit()
}

func TestEprobeHandler(t *testing.T) {
	signald := signal.NewSignalHandle(nil)
	ctx := signald.GetContext()

	serve := &http.Server{
		Addr:         ":8082",
		Handler:      http.NewServeMux(),
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
	go serve.ListenAndServe()
	New(ctx, WithHandler(serve.Handler.(*http.ServeMux))).Detect(&probe{})
	signald.WaitExit()
}
