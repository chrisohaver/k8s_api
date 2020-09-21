package podnames

import (
	"github.com/coredns/coredns/plugin"
	"k8s.io/client-go/tools/cache"
)

const pluginName = "k8s_api"

type PodNames struct {
	Next       plugin.Handler
	Zones      []string
	podIndexer cache.Indexer
	ttl uint32
}
