package kubernetes

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	k8sapi "github.com/coredns/coredns/plugin/k8s_api"
	"github.com/coredns/coredns/plugin/kubernetes/object"

	api "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

const (
	podIPIndex            = "PodIP"
	svcNameNamespaceIndex = "NameNamespace"
	svcIPIndex            = "ServiceIP"
	epNameNamespaceIndex  = "EndpointNameNamespace"
	epIPIndex             = "EndpointsIP"
)

type dnsController interface {
	ServiceList() []*object.Service
	EndpointsList() []*object.Endpoints
	SvcIndex(string) []*object.Service
	SvcIndexReverse(string) []*object.Service
	PodIndex(string) []*object.Pod
	EpIndex(string) []*object.Endpoints
	EpIndexReverse(string) []*object.Endpoints

	GetNamespaceByName(string) (*api.Namespace, error)

	HasSynced() bool

	// Modified returns the timestamp of the most recent changes
	Modified() int64

	// SetLister sets a lister for a the given name
	SetLister(name string, lister cache.KeyListerGetter) error
}

type dnsControl struct {
	// Modified tracks timestamp of the most recent changes
	// It needs to be first because it is guaranteed to be 8-byte
	// aligned ( we use sync.LoadAtomic with this )
	modified int64

	svcLister cache.Indexer
	podLister cache.Indexer
	epLister  cache.Indexer
	nsLister  cache.Store

	syncedFn k8sapi.HasSyncedFunc
}

type dnsControlOpts struct {
	initPodCache       bool
	initEndpointsCache bool
	ignoreEmptyService bool

	// Label handling.
	labelSelector          *meta.LabelSelector
	selector               labels.Selector
	namespaceLabelSelector *meta.LabelSelector
	namespaceSelector      labels.Selector

	zones                 []string
	endpointNameMode      bool
	skipAPIObjectsCleanup bool
}

// SetLister sets the named object lister to lister
func (dns *dnsControl) SetLister(name string, lister cache.KeyListerGetter) error {
	switch name {
	case "service":
		l, ok := lister.(cache.Indexer)
		if !ok {
			return fmt.Errorf("expected Indexer, got %v", lister)
		}
		dns.svcLister = l
	case "pod":
		l, ok := lister.(cache.Indexer)
		if !ok {
			return fmt.Errorf("expected Indexer, got %v", lister)
		}
		dns.podLister = l
	case "endpoints":
		l, ok := lister.(cache.Indexer)
		if !ok {
			return fmt.Errorf("expected Indexer, got %v", lister)
		}
		dns.epLister = l
	case "namespace":
		l, ok := lister.(cache.Store)
		if !ok {
			return fmt.Errorf("expected Store, got %v", lister)
		}
		dns.nsLister = l
	}

	return nil
}

func (dns *dnsControl) recordDNSProgrammingLatency(obj meta.Object) {
	recordDNSProgrammingLatency(dns.getServices(obj.(*api.Endpoints)), obj.(*api.Endpoints))
}

// HasSynced returns the sync status
func (dns *dnsControl) HasSynced() bool {
	if dns.syncedFn == nil {
		return false
	}
	return dns.syncedFn()
}

func (dns *dnsControl) ServiceList() (svcs []*object.Service) {
	os := dns.svcLister.List()
	for _, o := range os {
		s, ok := o.(*object.Service)
		if !ok {
			continue
		}
		svcs = append(svcs, s)
	}
	return svcs
}

func (dns *dnsControl) EndpointsList() (eps []*object.Endpoints) {
	os := dns.epLister.List()
	for _, o := range os {
		ep, ok := o.(*object.Endpoints)
		if !ok {
			continue
		}
		eps = append(eps, ep)
	}
	return eps
}

func (dns *dnsControl) PodIndex(ip string) (pods []*object.Pod) {
	os, err := dns.podLister.ByIndex(podIPIndex, ip)
	if err != nil {
		return nil
	}
	for _, o := range os {
		p, ok := o.(*object.Pod)
		if !ok {
			continue
		}
		pods = append(pods, p)
	}
	return pods
}

func (dns *dnsControl) SvcIndex(idx string) (svcs []*object.Service) {
	os, err := dns.svcLister.ByIndex(svcNameNamespaceIndex, idx)
	if err != nil {
		return nil
	}
	for _, o := range os {
		s, ok := o.(*object.Service)
		if !ok {
			continue
		}
		svcs = append(svcs, s)
	}
	return svcs
}

func (dns *dnsControl) SvcIndexReverse(ip string) (svcs []*object.Service) {
	os, err := dns.svcLister.ByIndex(svcIPIndex, ip)
	if err != nil {
		return nil
	}

	for _, o := range os {
		s, ok := o.(*object.Service)
		if !ok {
			continue
		}
		svcs = append(svcs, s)
	}
	return svcs
}

func (dns *dnsControl) EpIndex(idx string) (ep []*object.Endpoints) {
	os, err := dns.epLister.ByIndex(epNameNamespaceIndex, idx)
	if err != nil {
		return nil
	}
	for _, o := range os {
		e, ok := o.(*object.Endpoints)
		if !ok {
			continue
		}
		ep = append(ep, e)
	}
	return ep
}

func (dns *dnsControl) EpIndexReverse(ip string) (ep []*object.Endpoints) {
	os, err := dns.epLister.ByIndex(epIPIndex, ip)
	if err != nil {
		return nil
	}
	for _, o := range os {
		e, ok := o.(*object.Endpoints)
		if !ok {
			continue
		}
		ep = append(ep, e)
	}
	return ep
}

// GetNamespaceByName returns the namespace by name. If nothing is found an error is returned.
func (dns *dnsControl) GetNamespaceByName(name string) (*api.Namespace, error) {
	os := dns.nsLister.List()
	for _, o := range os {
		ns, ok := o.(*api.Namespace)
		if !ok {
			continue
		}
		if name == ns.ObjectMeta.Name {
			return ns, nil
		}
	}
	return nil, fmt.Errorf("namespace not found")
}

func (dns *dnsControl) Add(obj interface{})               { dns.updateModifed() }
func (dns *dnsControl) Delete(obj interface{})            { dns.updateModifed() }
func (dns *dnsControl) Update(oldObj, newObj interface{}) { dns.detectChanges(oldObj, newObj) }

// detectChanges detects changes in objects, and updates the modified timestamp
func (dns *dnsControl) detectChanges(oldObj, newObj interface{}) {
	// If both objects have the same resource version, they are identical.
	if newObj != nil && oldObj != nil && (oldObj.(meta.Object).GetResourceVersion() == newObj.(meta.Object).GetResourceVersion()) {
		return
	}
	obj := newObj
	if obj == nil {
		obj = oldObj
	}
	switch ob := obj.(type) {
	case *object.Service:
		dns.updateModifed()
	case *object.Pod:
		dns.updateModifed()
	case *object.Endpoints:
		if !endpointsEquivalent(oldObj.(*object.Endpoints), newObj.(*object.Endpoints)) {
			dns.updateModifed()
		}
	default:
		log.Warningf("Updates for %T not supported.", ob)
	}
}

func (dns *dnsControl) getServices(endpoints *api.Endpoints) []*object.Service {
	return dns.SvcIndex(object.EndpointsKey(endpoints.GetName(), endpoints.GetNamespace()))
}

// subsetsEquivalent checks if two endpoint subsets are significantly equivalent
// I.e. that they have the same ready addresses, host names, ports (including protocol
// and service names for SRV)
func subsetsEquivalent(sa, sb object.EndpointSubset) bool {
	if len(sa.Addresses) != len(sb.Addresses) {
		return false
	}
	if len(sa.Ports) != len(sb.Ports) {
		return false
	}

	// in Addresses and Ports, we should be able to rely on
	// these being sorted and able to be compared
	// they are supposed to be in a canonical format
	for addr, aaddr := range sa.Addresses {
		baddr := sb.Addresses[addr]
		if aaddr.IP != baddr.IP {
			return false
		}
		if aaddr.Hostname != baddr.Hostname {
			return false
		}
	}

	for port, aport := range sa.Ports {
		bport := sb.Ports[port]
		if aport.Name != bport.Name {
			return false
		}
		if aport.Port != bport.Port {
			return false
		}
		if aport.Protocol != bport.Protocol {
			return false
		}
	}
	return true
}

// endpointsEquivalent checks if the update to an endpoint is something
// that matters to us or if they are effectively equivalent.
func endpointsEquivalent(a, b *object.Endpoints) bool {
	if a == nil || b == nil {
		return false
	}

	if len(a.Subsets) != len(b.Subsets) {
		return false
	}

	// we should be able to rely on
	// these being sorted and able to be compared
	// they are supposed to be in a canonical format
	for i, sa := range a.Subsets {
		sb := b.Subsets[i]
		if !subsetsEquivalent(sa, sb) {
			return false
		}
	}
	return true
}

func (dns *dnsControl) Modified() int64 {
	unix := atomic.LoadInt64(&dns.modified)
	return unix
}

// updateModified set dns.modified to the current time.
func (dns *dnsControl) updateModifed() {
	unix := time.Now().Unix()
	atomic.StoreInt64(&dns.modified, unix)
}

var errObj = errors.New("obj was not of the correct type")

const defaultResyncPeriod = 0
