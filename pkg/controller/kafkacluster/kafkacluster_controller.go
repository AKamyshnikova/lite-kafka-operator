package kafkacluster

import (
	"context"

	litekafkav1alpha1 "github.com/Svimba/lite-kafka-operator/pkg/apis/litekafka/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_kafkacluster")

// Add creates a new KafkaCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKafkaCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kafkacluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KafkaCluster
	err = c.Watch(&source.Kind{Type: &litekafkav1alpha1.KafkaCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner KafkaCluster
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &litekafkav1alpha1.KafkaCluster{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Service and requeue the owner KafkaCluster
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &litekafkav1alpha1.KafkaCluster{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StateulSet and requeue the owner KafkaCluster
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &litekafkav1alpha1.KafkaCluster{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKafkaCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKafkaCluster{}

// ReconcileKafkaCluster reconciles a KafkaCluster object
type ReconcileKafkaCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	kafka  *litekafkav1alpha1.KafkaCluster
	rlog   logr.Logger
}

// Reconcile reads that state of the cluster for a KafkaCluster object and makes changes based on the state read
// and what is in the KafkaCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKafkaCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r.rlog = log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	r.rlog.Info("Reconciling KafkaCluster")

	// Fetch the KafkaCluster instance
	r.kafka = &litekafkav1alpha1.KafkaCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, r.kafka)
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
		ready, err := CheckZookeeperIsReady(r.kafka.Spec.Zookeeper.Host, r.kafka.Spec.Zookeeper.Port.Port)
		if err != nil {
			r.rlog.Error(err, "Error during testing Zookeeper service")
			return reconcile.Result{Requeue: false}, err
		}
		if !ready {
			r.rlog.Info("Zookeeper service is not ready, reconcile")
			return reconcile.Result{Requeue: true}, nil
		}
		r.rlog.Info("Zookeeper service is ready, continue to deploy resources")
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
	return reconcile.Result{}, nil
}
