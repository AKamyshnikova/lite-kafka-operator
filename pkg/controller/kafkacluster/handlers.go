package kafkacluster

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileKafkaCluster) handleSTSKafka() (bool, error) {
	// Define a new object
	obj, err := getKafkaStatefulSet(r.kafka)
	if err != nil {
		return false, err
	}

	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, obj, r.scheme); err != nil {
		return false, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.rlog.Info("Creating a new StatefulSet", "Namespace", obj.Namespace, "Name", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			return false, err
		}

		// Pod created successfully - don't requeue
		return false, nil
	} else if err != nil {
		return false, err
	}

	// Pod already exists - don't requeue
	r.rlog.Info("Skip reconcile: StatefulSet already exists", "Namespace", found.Namespace, "Name", found.Name)
	return false, nil
}
