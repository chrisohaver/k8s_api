package k8sapi

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type APIWatcher interface {

	// Informers should return a list of functions that return an Kubernetes Object Informer, each mapped by watch name.
	// k8s_api will start each Informer on the API connection. If multiple plugins return an Informer func with the same
	// name, the first plugin (per plugin execution order), will take precedence.
	Informers() map[string]InformerFunc

	// SetIndexer should set the index passed to a local pointer to be used by the plugin.  k8s_api calls this function
	// for *all* Informers added via the Informers() function by any plugins implementing APIWatcher. This enables
	// multiple plugins to share the same store/index managed by a single Informer.
	SetIndexer(string, cache.KeyListerGetter) error

	// SetHasSynced should set the HasSyncedFunc passed to a local function to be used by the plugin. Implement this
	// if the plugin needs to know if the API informers have fully synced (i.e. completed initial list action before
	// beginning the watch).
	SetHasSynced(HasSyncedFunc)
}

type HasSyncedFunc func() bool

type InformerFunc func(context.Context, kubernetes.Interface) *Informer
