package podnames

import (
	"strconv"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"       // pull this in here, because we want it excluded if plugin.cfg doesn't have k8s
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"      // pull this in here, because we want it excluded if plugin.cfg doesn't have k8s
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack" // pull this in here, because we want it excluded if plugin.cfg doesn't have k8s
)


var log = clog.NewWithPlugin(pluginName)

func init() { plugin.Register(pluginName, setup) }

func setup(c *caddy.Controller) error {

	p, err := parse(c)
	if err != nil {
		return plugin.Error(pluginName, err)
	}


	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		return p
	})

	return nil
}

func parse(c *caddy.Controller) (*PodNames, error) {
	var (
		p *PodNames
		err error
	)

	i := 0
	for c.Next() {
		if i > 0 {
			return nil, plugin.ErrOnce
		}
		i++

		p, err = parseStanza(c)
		if err != nil {
			return p, err
		}
	}
	return p, nil
}

func parseStanza(c *caddy.Controller) (*PodNames, error) {
	p := &PodNames{ttl: 5}

	zones := c.RemainingArgs()
	if len(zones) != 0 {
		p.Zones = zones
		for i := 0; i < len(p.Zones); i++ {
			p.Zones[i] = plugin.Host(p.Zones[i]).Normalize()
		}
	} else {
		// inherit zones from server block
		p.Zones = make([]string, len(c.ServerBlockKeys))
		for i := 0; i < len(c.ServerBlockKeys); i++ {
			p.Zones[i] = plugin.Host(c.ServerBlockKeys[i]).Normalize()
		}
	}

	// TODO check that first zone is not a reverse zone?

	for c.NextBlock() {
		switch c.Val() {
		case "ttl":
			args := c.RemainingArgs()
			if len(args) == 0 {
				return nil, c.ArgErr()
			}
			t, err := strconv.Atoi(args[0])
			if err != nil {
				return nil, err
			}
			if t < 0 || t > 3600 {
				return nil, c.Errf("ttl must be in range [0, 3600]: %d", t)
			}
			p.ttl = uint32(t)
		default:
			return nil, c.Errf("unknown property '%s'", c.Val())
		}
	}

	return p, nil
}
