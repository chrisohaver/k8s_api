# podnames

## Name

*podnames* - Serve A/AAAA/PTR records for Pods by Pod Name.

## Description

Enables Pod lookup by pod name/namespace. e.g. `mypod.mynamespace.mydomain.`.
  
This does not follow the [Kubernetes DNS-Based Service Discovery
Specification](https://github.com/kubernetes/dns/blob/master/docs/specification.md).

This plugin requires the *k8s_api* and companion *kubernetes* plugin in 
https://github.com/chrisohaver/k8s_api/tree/master/examples.  The *kubernetes*
plugin must have the `pods verified` option.

## Syntax

```
podnames [ZONES...] {
    ttl TTL
}
```

* `ttl` allows you to set a custom TTL for responses. The default is 5 seconds.  The minimum TTL allowed is
  0 seconds, and the maximum is capped at 3600 seconds. Setting TTL to 0 will prevent records from being cached.

## Ready

This plugin reports readiness to the ready plugin. This will happen after the
Kubernetes API connection is synced.

## Examples

Create records for Pods by pod name in the domain `pod.cluster.local.`
e.g. `mypod.mynamespace.pod.cluster.local.`.  This example eclipses the
existing ip based `pod.cluster.local.` records that *kubernetes* plugin
creates.

~~~ txt
  .:53 {
    podnames pod.cluster.local in-addr.arpa ip6.arpa

    kubernetes cluster.local in-addr.arpa ip6.arpa {
      pods verified
    }

    k8s_api
  }
~~~
