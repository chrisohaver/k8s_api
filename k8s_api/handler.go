package k8sapi

import (
	"context"

	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Handler interface.
func (k KubeAPI) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
		return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
}

// Name implements the plugin.Handler interface.
func (k KubeAPI) Name() string { return pluginName }
