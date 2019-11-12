package kafkacluster

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *ReconcileKafkaCluster) handleSTSKafka() (bool, error) {
	// Define a new object
	obj := getKafkaStatefulSet(r.kafka)

	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, obj, r.scheme); err != nil {
		return false, err
	}

	// Check if this Pod already exists
	found := &appsv1.StatefulSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
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

	r.rlog.Info("Check replicas of StatefulSet", "Namespace", obj.Namespace, "Name", obj.Name)
	// Check replicas
	if *found.Spec.Replicas != *obj.Spec.Replicas {
		found.Spec.Replicas = obj.Spec.Replicas
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			r.rlog.Error(err, "Cannot update number of replicas for StatefulSet")
			return true, err
		}

	}

	// Pod already exists - don't requeue
	r.rlog.Info("Skip reconcile: StatefulSet already exists", "Namespace", found.Namespace, "Name", found.Name)
	return false, nil
}

func (r *ReconcileKafkaCluster) handleSVCsKafka() (bool, error) {
	// Define a new object
	objHeadless := getKafkaServiceHeadless(r.kafka)

	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, objHeadless, r.scheme); err != nil {
		return false, err
	}

	// Check if this Service already exists
	foundHeadless := &corev1.Service{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: objHeadless.Name, Namespace: objHeadless.Namespace}, foundHeadless)
	if err != nil && errors.IsNotFound(err) {
		r.rlog.Info("Creating a new Service", "Namespace", objHeadless.Namespace, "Name", objHeadless.Name)
		err = r.client.Create(context.TODO(), objHeadless)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		// Pod already exists - continue to next svc
		r.rlog.Info("Skip reconcile: Service already exists", "Namespace", foundHeadless.Namespace, "Name", foundHeadless.Name)
	}

	// Define a new object
	obj := getKafkaService(r.kafka)
	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, obj, r.scheme); err != nil {
		return false, err
	}
	// Check if this Service already exists
	found := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.rlog.Info("Creating a new Service", "Namespace", obj.Namespace, "Name", obj.Name)
		err = r.client.Create(context.TODO(), obj)
		if err != nil {
			return false, err
		}

		// Pod created successfully - don't requeue
		return false, nil
	} else if err != nil {
		return false, err
	}
	r.rlog.Info("Skip reconcile: Service already exists", "Namespace", found.Namespace, "Name", found.Name)

	return false, nil
}
