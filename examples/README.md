# Examples

* examples/kubernetes - is a copy of CoreDNS's built-in kubernetes plugin (circa 1.7.0),
refactored to use the k8s_api plugin.  It registers 4 informers: "service",
"endpoints", "namespace", and "pod" (if `pods verified` is declared).

* examples/podnames - Work in Progress - enable pod lookup by 
  podname/namespace - e.g. `mypod.mynamespace.pod.cluster.local`