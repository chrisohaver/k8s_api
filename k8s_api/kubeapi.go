// Package kubernetes provides the kubernetes backend.
package k8sapi

import (
	"context"
	"fmt"

	"github.com/coredns/coredns/plugin"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/labels"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// KubeAPI implements a plugin that connects to a Kubernetes cluster.
type KubeAPI struct {
	Next plugin.Handler

	APIServer     string
	APICertAuth   string
	APIClientCert string
	APIClientKey  string
	ClientConfig  clientcmd.ClientConfig
	APIConn       apiController

	namespaces map[string]struct{}
	selector labels.Selector
	nsSelector labels.Selector
}

// New returns a initialized Kubernetes. It default interfaceAddrFunc to return 127.0.0.1. All other
// values default to their zero value, primaryZoneIndex will thus point to the first zone.
func New(zones []string) *KubeAPI {
	k := new(KubeAPI)
	return k
}

func (k *KubeAPI) getClientConfig() (*rest.Config, error) {
	if k.ClientConfig != nil {
		return k.ClientConfig.ClientConfig()
	}
	loadingRules := &clientcmd.ClientConfigLoadingRules{}
	overrides := &clientcmd.ConfigOverrides{}
	clusterinfo := clientcmdapi.Cluster{}
	authinfo := clientcmdapi.AuthInfo{}

	// Connect to API from in cluster if APIServer is not specified
	if k.APIServer  == "" {
		cc, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		cc.ContentType = "application/vnd.kubernetes.protobuf"
		return cc, err
	}

	// Connect to API from out of cluster
	clusterinfo.Server = k.APIServer

	if len(k.APICertAuth) > 0 {
		clusterinfo.CertificateAuthority = k.APICertAuth
	}
	if len(k.APIClientCert) > 0 {
		authinfo.ClientCertificate = k.APIClientCert
	}
	if len(k.APIClientKey) > 0 {
		authinfo.ClientKey = k.APIClientKey
	}

	overrides.ClusterInfo = clusterinfo
	overrides.AuthInfo = authinfo
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	cc, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	cc.ContentType = "application/vnd.kubernetes.protobuf"
	return cc, err

}

// InitKubeCache initializes a new Kubernetes cache.
func (k *KubeAPI) InitKubeCache(ctx context.Context) (err error) {
	config, err := k.getClientConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes notification controller: %q", err)
	}

	k.APIConn = &apiControl{
		client:            kubeClient,
		stopCh:            make(chan struct{}),
	}

	return err
}
