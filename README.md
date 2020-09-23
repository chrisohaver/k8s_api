# k8s_api

## Name

*k8s_api* - a CoreDNS plugin that enables other plugins to register/share Kubernetes API watches.

## Description

*k8s_api* enables any plugin that implementing the `k8sapi.APIWatcher` interface, to register Kubernetes API informers, and share access to object stores.

```
type APIWatcher interface {

	// Informers should return a list of functions that return an Kubernetes Object Informer, each mapped by watch name.
	// k8s_api will start each Informer on the API connection. If multiple plugins return an Informer func with the same
	// name, the first plugin (per plugin execution order), will take precedence.
	Informers() map[string]InformerFunc

	// SetIndexer should set the index passed to a local pointer to be used by the plugin.  k8s_api calls this function
	// for *all* Informers added via the Informers() function by any plugins implementing APIWatcher. This enables
	// multiple plugins to share the same store/index managed by a single Informer.
	SetIndexer(string, cache.KeyListerGetter) error

	// SetHasSynced should set the HasSyncedFunc passed to a local function to be used by the plugin. Implement this
	// if the plugin needs to know if the API informers have fully synced (i.e. completed initial list action before
	// beginning the watch).
	SetHasSynced(HasSyncedFunc)
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

* `endpoint` specifies the **URL** for a remote k8s API endpoint.
   If omitted, it will connect to k8s in-cluster using the cluster service account.
* `tls` **CERT** **KEY** **CACERT** are the TLS cert, key and the CA cert file names for remote k8s connection.
   This option is ignored if connecting in-cluster (i.e. endpoint is not specified).
* `kubeconfig` **KUBECONFIG** **CONTEXT** authenticates the connection to a remote k8s cluster using a kubeconfig file. It supports TLS, username and password, or token-based authentication. This option is ignored if connecting in-cluster (i.e., the endpoint is not specified).

## External Plugin

*k8s_api* is an *external* plugin, which means it is not included in CoreDNS releases.  To use *k8s_api*, you'll need to build a CoreDNS image with *k8s_api*. In a nutshell you'll need to:
* Clone https://github.com/coredns/coredns into `$GOPATH/src/github.com/coredns`
* Add this plugin to [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg) 
  per instructions therein. This plugin must be ordered *after* any plugins that use
  it (i.e. plugins that implement APIWatcher).
* `make -f Makefile.release DOCKER=your-docker-repo release`
* `make -f Makefile.release DOCKER=your-docker-repo docker`
* `make -f Makefile.release DOCKER=your-docker-repo docker-push`

## Examples

Example plugins that implement `k8sapi.APIWatcher` can be found in the `examples` directory. 

## TO-DOs

* Namespace filtering:

  * `namespaces` **NAMESPACE [NAMESPACE...]** only exposes the k8s namespaces listed.
   If this option is omitted all namespaces are exposed
   
* Object/namespace *label* filtering:

  * `namespace_labels` **EXPRESSION** only expose the records for Kubernetes namespaces that match this label selector.
   The label selector syntax is described in the
   [Kubernetes User Guide - Labels](https://kubernetes.io/docs/user-guide/labels/). An example that
   only exposes namespaces labeled as "istio-injection=enabled", would use:
   `labels istio-injection=enabled`.
  * `labels` **EXPRESSION** only exposes the records for Kubernetes objects that match this label selector.
   The label selector syntax is described in the
   [Kubernetes User Guide - Labels](https://kubernetes.io/docs/user-guide/labels/). An example that
   only exposes objects labeled as "application=nginx" in the "staging" or "qa" environments, would
   use: `labels environment in (staging, qa),application=nginx`.
