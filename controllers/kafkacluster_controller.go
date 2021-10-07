/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	litekafkav1alpha1 "github.com/Svimba/lite-kafka-operator/api/v1alpha1"
)

var log = logf.Log.WithName("controller_kafkacluster")

// KafkaClusterReconciler reconciles a KafkaCluster object
type KafkaClusterReconciler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	Client client.Client
	Scheme *runtime.Scheme
	kafka  *litekafkav1alpha1.KafkaCluster
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=litekafka.operator.mirantis.com,resources=kafkaclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=litekafka.operator.mirantis.com,resources=kafkaclusters/status,verbs=get;update;patch

func (r *KafkaClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	r.Log = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.Log.Info("Reconciling KafkaCluster")

	// Fetch the KafkaCluster instance
	r.kafka = &litekafkav1alpha1.KafkaCluster{}
	err := r.Client.Get(ctx, request.NamespacedName, r.kafka)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// set default values for undefined specs
	r.kafka.SetDefaults()

	// Check zookeeper service is ready
	if *r.kafka.Spec.ZookeeperCheck {
		r.Log.Info("Waiting until Zookeeper service is ready")
		ready, err := CheckZookeeperIsReady(r.kafka.Spec.Zookeeper.Host, r.kafka.Spec.Zookeeper.Port.Port)
		if err != nil {
			r.Log.Error(err, "Error during testing Zookeeper service")
			return reconcile.Result{Requeue: false}, err
		}
		if !ready {
			r.Log.Info("Zookeeper service is not ready, reconcile")
			return reconcile.Result{Requeue: true}, nil
		}
		r.Log.Info("Zookeeper service is ready, continue to deploy resources")
	}

	// Start resourec handling
	requeue, err := r.handleSTSKafka()
	if err != nil {
		return reconcile.Result{Requeue: requeue}, err
	}

	requeue, err = r.handleSVCsKafka()
	if err != nil {
		return reconcile.Result{Requeue: requeue}, err
	}

	requeue, err = r.UpdateClusterStatus()
	if err != nil {
		return reconcile.Result{Requeue: requeue}, err
	}

	requeue, err = r.handleConfigMap()
	if err != nil {
		return reconcile.Result{Requeue: requeue}, err
	}

	return reconcile.Result{}, nil
}

func (r *KafkaClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&litekafkav1alpha1.KafkaCluster{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Pod{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(r)
}
