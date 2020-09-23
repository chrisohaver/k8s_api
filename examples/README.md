# Examples

* examples/kubernetes - is a copy of CoreDNS's built-in kubernetes plugin (circa 1.7.0),
  refactored to use the k8s_api plugin.  It registers 4 informers: "service",
  "endpoints", "namespace", and "pod" (if `pods verified` is declared).

* examples/podnames - *Work in Progress* - enable pod lookup by 
  podname/namespace - e.g. `mypod.mynamespace.pod.cluster.local`.  It uses
  the "pod" informer created by the examples/kubernetes plugin above. e.g.
  
  plugin.cfg:
  ```
  ...
  podnames:github.com/chrisohaver/k8s_api/examples/podnames
  kubernetes:github.com/chrisohaver/k8s_api/examples/kubernetes
  k8s_api:github.com/chrisohaver/k8s_api/k8s_api
  ...
  ```
  Corefile:
  ```
  .:53 {
    podnames pod.cluster.local in-addr.arpa ip6.arpa {
      ttl 5
    }

    kubernetes cluster.local in-addr.arpa ip6.arpa {
      pods verified
    }

    k8s_api
  }

  ```