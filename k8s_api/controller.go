package k8sapi

import (
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type apiController interface {
	Run()
	HasSynced() bool
	Stop() error
}

type apiControl struct {
	client kubernetes.Interface

	Informers map[string]*Informer

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock sync.Mutex
	shutdown bool
	stopCh   chan struct{}

	zones            []string
	endpointNameMode bool
}

type Informer struct {
	Controller cache.Controller
	Lister cache.KeyListerGetter
}

// Stop stops the  controller.
func (dns *apiControl) Stop() error {
	dns.stopLock.Lock()
	defer dns.stopLock.Unlock()

	// Only try draining the workqueue if we haven't already.
	if !dns.shutdown {
		close(dns.stopCh)
		dns.shutdown = true
		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

// Run starts the controller.
func (dns *apiControl) Run() {
	for _, w := range dns.Informers {
		go w.Controller.Run(dns.stopCh)
	}
	<-dns.stopCh
}

// HasSynced calls on all controllers.
func (dns *apiControl) HasSynced() bool {
	for _, w := range dns.Informers {
		if !w.Controller.HasSynced() {
			return false
		}
	}
	return true
}