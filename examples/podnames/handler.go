package podnames

import (
	"context"
	"errors"
	"net"
	"strings"

	"github.com/chrisohaver/k8s_api/examples/kubernetes/object"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/dnsutil"
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

	if dnsutil.IsReverse(qname) > 0 {
		// handle reverse
		ip := dnsutil.ExtractAddressFromReverse(state.Name())
		objs, err := p.podIndexer.ByIndex("PodIP", ip) // expose constant from kubernetes package?
		if err != nil {
			return dns.RcodeServerFailure, err
		}
		m := &dns.Msg{}
		m.SetReply(r)
		for _, o := range objs {
			pod, ok := o.(object.Pod)
			if !ok {
				continue
			}
			m.Answer = append(m.Answer, &dns.PTR{
				Hdr: dns.RR_Header{
					Name:   qname,
					Class: dns.ClassINET,
					Rrtype: dns.TypePTR,
					Ttl:    p.ttl,
				},
				Ptr: pod.Name + "." + pod.Namespace + "." + p.Zones[0]})
		}
		err = w.WriteMsg(m)
		if err != nil {
			return dns.RcodeServerFailure, err
		}

		return dns.RcodeSuccess, nil
	}

	// strip zone off to get name/namespace
	sub := qname[0 : strings.Index(qname, zone)-1]
	segs := strings.Split(sub, ".")
	if len(segs) < 2 {
		return dns.RcodeNameError, nil
	}
	item, exists, err := p.podIndexer.GetByKey(segs[1] + "/" + segs[0])
	if err != nil {
		return dns.RcodeServerFailure, err
	}
	if !exists {
		return dns.RcodeNameError, nil
	}
	pod, ok := item.(*object.Pod)
	if !ok {
		return dns.RcodeServerFailure, errors.New("unexpected indexer item type")
	}
	ip := net.ParseIP(pod.PodIP)

	// construct the reply
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	if ip.To4() != nil && state.QType() == dns.TypeA {
		m.Answer = append(m.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   qname,
				Class: dns.ClassINET,
				Rrtype: dns.TypeA,
				Ttl:    p.ttl,
			},
			A: ip})
	}
	if ip.To16() != nil && state.QType() == dns.TypeAAAA {
		m.Answer = append(m.Answer, &dns.AAAA{
			Hdr: dns.RR_Header{
				Name:   qname,
				Class: dns.ClassINET,
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
