// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package discover

import (
	"reflect"
	"time"

	wraperrors "github.com/pkg/errors"
	"k8s.io/api/apps/v1beta1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"github.com/kenjones-cisco/dapperdox/discover/models"
)

// watcher defines an event loop.
//
// The watcher guarantees the following consistency requirement: registry
// view in the watcher is as AT LEAST as fresh as the moment notification
// arrives, but MAY BE more fresh (e.g. "delete" cancels an "add" event).  For
// example, an event for a service creation will see a service registry without
// the service if the event is immediately followed by the service deletion
// event.
//
// Handlers execute on the single worker queue in the order they are appended.
// Handlers receive the notification event and the associated object.  Note
// that all handlers must be appended before starting the watcher.
type watcher interface {
	// AppendServiceHandler notifies about changes to the service catalog.
	AppendServiceHandler(f func(*models.Service, models.Event)) error

	// AppendDeploymentHandler notifies about change to the deployment catalog.
	AppendDeploymentHandler(f func(*models.Deployment, models.Event)) error

	// Run until a signal is received
	Run(stop <-chan struct{})
}

// catalogOptions stores the configurable attributes of a cagalog.
type catalogOptions struct {
	// Namespace the controller watches. If set to meta_v1.NamespaceAll (""), controller watches all namespaces
	WatchedNamespace string
	ResyncPeriod     time.Duration
	DomainSuffix     string
}

// catalog is a collection of synchronized resource watchers
// caches are thread-safe.
type catalog struct {
	domainSuffix string

	client      kubernetes.Interface
	queue       Queue
	services    cacheHandler
	deployments cacheHandler
}

type cacheHandler struct {
	informer cache.SharedIndexInformer
	handler  *ChainHandler
}

// newClient creates an in cluster connection to Kubernetes API.
func newClient() (kubernetes.Interface, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// creates the clientset
	return kubernetes.NewForConfig(config)
}

// newCatalog creates a new Kubernetes controller.
func newCatalog(client kubernetes.Interface, options catalogOptions) watcher {
	log().Infof("Service controller watching namespace %q", options.WatchedNamespace)

	// Queue requires a time duration for a retry delay after a handler error
	out := &catalog{
		domainSuffix: options.DomainSuffix,
		client:       client,
		queue:        NewQueue(1 * time.Second),
	}

	out.services = out.createInformer(&v1.Service{}, options.ResyncPeriod,
		func(opts meta_v1.ListOptions) (runtime.Object, error) {
			return client.CoreV1().Services(options.WatchedNamespace).List(opts)
		},
		func(opts meta_v1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Services(options.WatchedNamespace).Watch(opts)
		})

	out.deployments = out.createInformer(&v1beta1.Deployment{}, options.ResyncPeriod,
		func(opts meta_v1.ListOptions) (runtime.Object, error) {
			return client.AppsV1beta1().Deployments(options.WatchedNamespace).List(opts)
		},
		func(opts meta_v1.ListOptions) (watch.Interface, error) {
			return client.AppsV1beta1().Deployments(options.WatchedNamespace).Watch(opts)
		})

	return out
}

// AppendServiceHandler implements a service catalog operation.
func (c *catalog) AppendServiceHandler(f func(*models.Service, models.Event)) error {
	c.services.handler.Append(func(obj interface{}, event models.Event) error {
		svc, _ := obj.(*v1.Service)

		log().Debugf("(Handler) Service Details: %v", *svc)

		// Do not handle "kube-system" services
		if svc.Namespace == meta_v1.NamespaceSystem {
			return nil
		}

		log().Debugf("Handle service %s in namespace %s", svc.Name, svc.Namespace)

		if svcConv := convertService(svc, c.domainSuffix); svcConv != nil {
			f(svcConv, event)
		}

		return nil
	})

	return nil
}

// AppendDeploymentHandler notifies about change to the deployment catalog.
func (c *catalog) AppendDeploymentHandler(f func(*models.Deployment, models.Event)) error {
	c.deployments.handler.Append(func(obj interface{}, event models.Event) error {
		dpl, _ := obj.(*v1beta1.Deployment)

		// Do not handle "kube-system" services
		if dpl.Namespace == meta_v1.NamespaceSystem {
			return nil
		}

		log().Debugf("(Handler) Deployment Details: %v", dpl.Status)

		// ensure there is at least one replica in ready state
		// in order to serve the API request to fetch API specs
		//
		// readiness works best with multiple instances deployed
		// single instance deployments would still rely on
		if dpl.Status.ReadyReplicas == 0 {
			log().Debug("no ready replicas, don't trigger event")

			return nil
		}

		log().Debugf("Handle deployment %s in namespace %s", dpl.Name, dpl.Namespace)

		if dplConv := convertDeployment(dpl); dplConv != nil {
			f(dplConv, event)
		}

		return nil
	})

	return nil
}

// Run all controllers until a signal is received.
func (c *catalog) Run(stop <-chan struct{}) {
	go c.queue.Run(stop)
	go c.services.informer.Run(stop)
	go c.deployments.informer.Run(stop)

	<-stop
	log().Info("watcher terminated")
}

// notify is the first handler in the handler chain.
// Returning an error causes repeated execution of the entire chain.
func (c *catalog) notify(obj interface{}, event models.Event) error {
	if !c.services.informer.HasSynced() {
		return wraperrors.New("waiting till full synchronization")
	}

	k, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		log().Errorf("Error retrieving key: %v", err)
	} else {
		log().Debugf("Event %s: key %#v", event, k)
	}

	return nil
}

func (c *catalog) createInformer(o runtime.Object, resyncPeriod time.Duration, lf cache.ListFunc, wf cache.WatchFunc) cacheHandler {
	handler := &ChainHandler{funcs: []Handler{c.notify}}

	// TODO: finer-grained index (perf)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{ListFunc: lf, WatchFunc: wf}, o,
		resyncPeriod, cache.Indexers{})

	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			// TODO: filtering functions to skip over un-referenced resources (perf)
			AddFunc: func(obj interface{}) {
				c.queue.Push(Task{handler: handler.Apply, obj: obj, event: models.EventAdd})
			},
			UpdateFunc: func(old, cur interface{}) {
				if !reflect.DeepEqual(old, cur) {
					c.queue.Push(Task{handler: handler.Apply, obj: cur, event: models.EventUpdate})
				}
			},
			DeleteFunc: func(obj interface{}) {
				c.queue.Push(Task{handler: handler.Apply, obj: obj, event: models.EventDelete})
			},
		})

	return cacheHandler{informer: informer, handler: handler}
}
