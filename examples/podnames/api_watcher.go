package podnames

import (
	"errors"

	k8sapi "github.com/chrisohaver/k8s_api/k8s_api"
	"k8s.io/client-go/tools/cache"
)

func (p *PodNames) Informers() map[string]k8sapi.InformerFunc { return nil }

func (p *PodNames) SetIndexer(name string, lister cache.KeyListerGetter) error {
	if name != "pod" {
		return nil
	}
	pidx, ok := lister.(cache.Indexer)
	if !ok {
		return errors.New("unexpected lister type")
	}
	p.podIndexer = pidx
	return nil
}

func (p *PodNames) SetHasSynced(syncedFunc k8sapi.HasSyncedFunc) {}
