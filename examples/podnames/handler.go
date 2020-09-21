package podnames

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/chrisohaver/k8s_api/examples/kubernetes/object"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
)

// ServeDNS implements the plugin.Handler interface.
func (p PodNames) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	if state.QType() != dns.TypeA && state.QType() != dns.TypeAAAA {
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}
	qname := state.QName()
	zone := plugin.Zones(p.Zones).Matches(qname)
	if zone == "" {
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}

	// strip zone off to get name/namespace
	sub := qname[0:strings.Index(qname, zone)]
	key := strings.Replace(sub,".","/", 1)
	item, exists, err := p.podIndexer.GetByKey(key)
	if err != nil {
		return dns.RcodeServerFailure, err
	}
	if !exists {
		// Fallthrough to next plugin if no match found
		return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	}
	pod, ok := item.(*object.Pod)
	if !ok {
		return dns.RcodeServerFailure, errors.New("unexpected indexer item type")
	}
	ip := net.ParseIP(pod.PodIP)

	// construct the reply
	m := &dns.Msg{}
	m.SetReply(r)
	if ip.To4() != nil && state.QType() == dns.TypeA {
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   qname,
				Rrtype: dns.TypeA,
				Ttl:    p.ttl,
			},
			A: ip})
	}
	if ip.To16() != nil && state.QType() == dns.TypeAAAA {
		m.Answer = append(m.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   qname,
				Rrtype: dns.TypeAAAA,
				Ttl:    p.ttl,
			},
			AAAA: ip})
	}

	// write reply
	err = w.WriteMsg(m)
	if err != nil {
		return dns.RcodeServerFailure, err
	}

	return dns.RcodeSuccess, nil
}

// Name implements the plugin.Handler interface.
func (p PodNames) Name() string { return pluginName }
