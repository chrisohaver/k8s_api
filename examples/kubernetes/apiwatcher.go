package kubernetes

import (
	"context"

	k8sapi "github.com/chrisohaver/k8s_api/k8s_api"
	"github.com/chrisohaver/k8s_api/examples/kubernetes/object"
	api "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func (k *Kubernetes) Informers() map[string]k8sapi.InformerFunc {
	infuncs := make(map[string]k8sapi.InformerFunc)

	infuncs["service"] = func(ctx context.Context, client kubernetes.Interface) *k8sapi.Informer {
		svcLister, svcController := object.NewIndexerInformer(
			&cache.ListWatch{
				ListFunc:  serviceListFunc(ctx, client, api.NamespaceAll, k.opts.selector),
				WatchFunc: serviceWatchFunc(ctx, client, api.NamespaceAll, k.opts.selector),
			},
			&api.Service{},
			cache.ResourceEventHandlerFuncs{
				AddFunc:    k.APIConn.(*dnsControl).Add,
				UpdateFunc: k.APIConn.(*dnsControl).Update,
				DeleteFunc: k.APIConn.(*dnsControl).Delete,
			},
			cache.Indexers{svcNameNamespaceIndex: svcNameNamespaceIndexFunc, svcIPIndex: svcIPIndexFunc},
			object.DefaultProcessor(object.ToService(k.opts.skipAPIObjectsCleanup), nil),
		)
		return &k8sapi.Informer{Controller: svcController, Lister: svcLister}
	}

	if k.opts.initPodCache {
		infuncs["pod"] = func(ctx context.Context, client kubernetes.Interface) *k8sapi.Informer {
			podLister, podController := object.NewIndexerInformer(
				&cache.ListWatch{
					ListFunc:  podListFunc(ctx, client, api.NamespaceAll, k.opts.selector),
					WatchFunc: podWatchFunc(ctx, client, api.NamespaceAll, k.opts.selector),
				},
				&api.Pod{},
				cache.ResourceEventHandlerFuncs{
					AddFunc:    k.APIConn.(*dnsControl).Add,
					UpdateFunc: k.APIConn.(*dnsControl).Update,
					DeleteFunc: k.APIConn.(*dnsControl).Delete,
				},
				cache.Indexers{podIPIndex: podIPIndexFunc},
				object.DefaultProcessor(object.ToPod(k.opts.skipAPIObjectsCleanup), nil),
			)
			return &k8sapi.Informer{Controller: podController, Lister: podLister}
		}
	}

	if k.opts.initEndpointsCache {
		infuncs["endpoints"] = func(ctx context.Context, client kubernetes.Interface) *k8sapi.Informer {
			epLister, epController := object.NewIndexerInformer(
				&cache.ListWatch{
					ListFunc:  endpointsListFunc(ctx, client, api.NamespaceAll, k.opts.selector),
					WatchFunc: endpointsWatchFunc(ctx, client, api.NamespaceAll, k.opts.selector),
				},
				&api.Endpoints{},
				cache.ResourceEventHandlerFuncs{
					AddFunc:    k.APIConn.(*dnsControl).Add,
					UpdateFunc: k.APIConn.(*dnsControl).Update,
					DeleteFunc: k.APIConn.(*dnsControl).Delete,
				},
				cache.Indexers{epNameNamespaceIndex: epNameNamespaceIndexFunc, epIPIndex: epIPIndexFunc},
				object.DefaultProcessor(object.ToEndpoints(k.opts.skipAPIObjectsCleanup), k.APIConn.(*dnsControl).recordDNSProgrammingLatency),
			)
			return &k8sapi.Informer{Controller: epController, Lister: epLister}
		}
	}

	infuncs["namespace"] = func(ctx context.Context, client kubernetes.Interface) *k8sapi.Informer {
		nsLister, nsController := cache.NewInformer(
			&cache.ListWatch{
				ListFunc:  namespaceListFunc(ctx, client, k.opts.namespaceSelector),
				WatchFunc: namespaceWatchFunc(ctx, client, k.opts.namespaceSelector),
			},
			&api.Namespace{},
			defaultResyncPeriod,
			cache.ResourceEventHandlerFuncs{})
		return &k8sapi.Informer{Controller: nsController, Lister: nsLister}
	}

	return infuncs
}

func (k *Kubernetes) SetIndexer(name string, lister cache.KeyListerGetter) error {
	return k.APIConn.SetLister(name, lister)
}

func (k *Kubernetes) SetHasSynced(syncedFunc k8sapi.HasSyncedFunc) {
	k.APIConn.(*dnsControl).syncedFn = syncedFunc
}

func podIPIndexFunc(obj interface{}) ([]string, error) {
	p, ok := obj.(*object.Pod)
	if !ok {
		return nil, errObj
	}
	return []string{p.PodIP}, nil
}

func svcIPIndexFunc(obj interface{}) ([]string, error) {
	svc, ok := obj.(*object.Service)
	if !ok {
		return nil, errObj
	}
	if len(svc.ExternalIPs) == 0 {
		return []string{svc.ClusterIP}, nil
	}

	return append([]string{svc.ClusterIP}, svc.ExternalIPs...), nil
}

func svcNameNamespaceIndexFunc(obj interface{}) ([]string, error) {
	s, ok := obj.(*object.Service)
	if !ok {
		return nil, errObj
	}
	return []string{s.Index}, nil
}

func epNameNamespaceIndexFunc(obj interface{}) ([]string, error) {
	s, ok := obj.(*object.Endpoints)
	if !ok {
		return nil, errObj
	}
	return []string{s.Index}, nil
}

func epIPIndexFunc(obj interface{}) ([]string, error) {
	ep, ok := obj.(*object.Endpoints)
	if !ok {
		return nil, errObj
	}
	return ep.IndexIP, nil
}

func serviceListFunc(ctx context.Context, c kubernetes.Interface, ns string, s labels.Selector) func(meta.ListOptions) (runtime.Object, error) {
	return func(opts meta.ListOptions) (runtime.Object, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		listV1, err := c.CoreV1().Services(ns).List(ctx, opts)
		return listV1, err
	}
}

func podListFunc(ctx context.Context, c kubernetes.Interface, ns string, s labels.Selector) func(meta.ListOptions) (runtime.Object, error) {
	return func(opts meta.ListOptions) (runtime.Object, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		if len(opts.FieldSelector) > 0 {
			opts.FieldSelector = opts.FieldSelector + ","
		}
		opts.FieldSelector = opts.FieldSelector + "status.phase!=Succeeded,status.phase!=Failed,status.phase!=Unknown"
		listV1, err := c.CoreV1().Pods(ns).List(ctx, opts)
		return listV1, err
	}
}

func endpointsListFunc(ctx context.Context, c kubernetes.Interface, ns string, s labels.Selector) func(meta.ListOptions) (runtime.Object, error) {
	return func(opts meta.ListOptions) (runtime.Object, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		listV1, err := c.CoreV1().Endpoints(ns).List(ctx, opts)
		return listV1, err
	}
}

func namespaceListFunc(ctx context.Context, c kubernetes.Interface, s labels.Selector) func(meta.ListOptions) (runtime.Object, error) {
	return func(opts meta.ListOptions) (runtime.Object, error) {
		if s != nil {
			opts.LabelSelector = s.String()
		}
		listV1, err := c.CoreV1().Namespaces().List(ctx, opts)
		return listV1, err
	}
}
