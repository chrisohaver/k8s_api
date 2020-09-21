# k8s_api

## Name

*k8s_api* - a CoreDNS plugin that enables other plugins to register/share Kubernetes API watches.

## Description

*k8s_api* enables any plugin that implementing the `k8sapi.APIWatcher` interface, to register Kubernetes API informers, and share access to object stores.

```
type APIWatcher interface {
  # Informers should return list of functions that return an Kubernetes Object Informer, each mapped by watch name. 
  # k8s_api will start each Informer on the API connection.
  Informers() map[string]InformerFunc 
  
  # SetIndexer should set the index passed to a local pointer to be used by the implementor.  k8s_api calls this function for *all* Informers 
  # added by the implementor and other implementors via the Informers() function. This enables multiple plugins/implementors to share the same
  # stores/indexes created by a single Informer.
  SetIndexer(string, cache.KeyListerGetter) error
  
  # SetHasSynced should set the HasSyncedFunc passed to a local function to be used by the implementor.  Implement this if the plugin needs to
  # know if the API informers have fully synced (i.e. completed initial list action before beginning the watch). 
  SetHasSynced(HasSyncedFunc) # If required, should set a local function to be used to check sync status of the watches.
}
```


## Syntax

```
k8s_api {
    endpoint URL
    tls CERT KEY CACERT
    kubeconfig KUBECONFIG CONTEXT
}

```


## External Plugin

*k8s_api* is an *external* plugin, which means it is not included in CoreDNS releases.  To use *k8s_api*, you'll need to build a CoreDNS image with *k8s_api*. In a nutshell you'll need to:
* Clone https://github.com/coredns/coredns into `$GOPATH/src/github.com/coredns`
* Add this plugin to [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg) per instructions therein.
* `make -f Makefile.release DOCKER=your-docker-repo release`
* `make -f Makefile.release DOCKER=your-docker-repo docker`
* `make -f Makefile.release DOCKER=your-docker-repo docker-push`

## Examples

To Do
