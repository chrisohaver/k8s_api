package k8sapi

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type APIWatcher interface {
	Informers() map[string]InformerFunc
	SetIndexer(string, cache.KeyListerGetter) error
	SetHasSynced(HasSyncedFunc)
}

type HasSyncedFunc func() bool
type InformerFunc func(context.Context, kubernetes.Interface) *Informer
