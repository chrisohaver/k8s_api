package k8sapi

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"k8s.io/client-go/kubernetes"

	"github.com/caddyserver/caddy"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const pluginName = "k8s_api"

var log = clog.NewWithPlugin(pluginName)

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {
	klog.SetOutput(os.Stdout)

	k, err := parse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}

	k.RegisterKubeCache(c)

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		k.Next = next
		return k
	})

	return nil
}

func (k *KubeAPI) getAPIWatchers(c *caddy.Controller) error {

	config, err := k.getClientConfig()
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return plugin.Error(pluginName, fmt.Errorf("failed to create kubernetes notification controller: %q", err))
	}

	// Get Informer functions from all plugins implementing Watcher
	informerFuncs := make(map[string]InformerFunc)
	plugins := dnsserver.GetConfig(c).Handlers()
	for _, pl := range plugins {
		if w, ok := pl.(APIWatcher); ok {
			ifn := w.Informers()
			for n, f := range ifn {
				// skip if an informer with same name already exists
				if _, ok := informerFuncs[n]; ok {
					continue
				}
				informerFuncs[n] = f
			}
		}
	}
	// Call Informer functions and save result to the api controller
	apicon := apiControl{
		client:    kubeClient,
		stopCh:    make(chan struct{}),
		Informers: make(map[string]*Informer, len(informerFuncs)),
	}
	for n, f := range informerFuncs {
		inf := f(context.Background(), kubeClient)
		apicon.Informers[n] = inf
	}
	// Call SetIndexer for each Informer and HasSynced in all plugins implementing Watcher
	for _, pl := range plugins {
		if w, ok := pl.(APIWatcher); ok {
			for n, i := range apicon.Informers {
				err := w.SetIndexer(n, i.Lister)
				if err != nil {
					return err
				}
			}
			w.SetHasSynced(func() bool {
				// return false if at least one controller is not yet synced
				for i := range w.Informers() {
					if !apicon.Informers[i].Controller.HasSynced() {
						return false
					}
				}
				return true
			})

		}
	}

	k.APIConn = &apicon
	return nil
}

// RegisterKubeCache registers KubeCache start and stop functions with Caddy
func (k *KubeAPI) RegisterKubeCache(c *caddy.Controller) {
	c.OnStartup(func() error {

		err := k.getAPIWatchers(c)
		if err != nil {
			return err
		}

		go k.APIConn.Run()

		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ticker.C:
				if k.APIConn.HasSynced() {
					return nil
				}
			case <-timeout:
				return nil
			}
		}
	})

	c.OnShutdown(func() error {
		return k.APIConn.Stop()
	})
}

func parse(c *caddy.Controller) (*KubeAPI, error) {
	var (
		kapi *KubeAPI
		err  error
	)
	i := 0
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++

		kapi, err = parseStanza(c)
		if err != nil {
			return kapi, err
		}
	}
	return kapi, nil
}

// parseStanza parses a k8s_api stanza
func parseStanza(c *caddy.Controller) (*KubeAPI, error) {
	kapi := New([]string{""})
	for c.NextBlock() {
		switch c.Val() {
		case "endpoint":
			args := c.RemainingArgs()
			if len(args) == 1 {
				kapi.APIServer = args[0]
				continue
			}
			return nil, c.ArgErr()
		case "tls": // cert key cacertfile
			args := c.RemainingArgs()
			if len(args) == 3 {
				kapi.APIClientCert, kapi.APIClientKey, kapi.APICertAuth = args[0], args[1], args[2]
				continue
			}
			return nil, c.ArgErr()
		case "kubeconfig":
			args := c.RemainingArgs()
			if len(args) == 2 {
				config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
					&clientcmd.ClientConfigLoadingRules{ExplicitPath: args[0]},
					&clientcmd.ConfigOverrides{CurrentContext: args[1]},
				)
				kapi.ClientConfig = config
				continue
			}
			return nil, c.ArgErr()
		case "namespaces":
			args := c.RemainingArgs()
			if len(args) > 0 {
				for _, a := range args {
					kapi.namespaces[a] = struct{}{}
				}
				continue
			}
			return nil, c.ArgErr()
		default:
			return nil, c.Errf("unknown property '%s'", c.Val())
		}
	}
	return kapi, nil
}
