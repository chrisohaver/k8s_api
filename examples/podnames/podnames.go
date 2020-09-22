package podnames

import (
	"github.com/coredns/coredns/plugin"
	"k8s.io/client-go/tools/cache"
)

const pluginName = "podnames"

type PodNames struct {
	Next       plugin.Handler
	Zones      []string
	podIndexer cache.Indexer
	ttl uint32
}
