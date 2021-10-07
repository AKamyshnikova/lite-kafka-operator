package controllers

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *KafkaClusterReconciler) handleSTSKafka() (bool, error) {
	// Define a new object
	obj := getKafkaStatefulSet(r.kafka)

	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, obj, r.Scheme); err != nil {
		return false, err
	}

	// Check old version of kafka StatefulSet via labels. There was different name for StatefulSet
	// then operator creates next STS which is in conflict with old one.
	err := r.deleteOldKafkaStatefulSet("app.kubernetes.io/name=kafka,app.kubernetes.io/component=kafka-broker")
	if err != nil {
		return false, err
	}

	// Check if this Pod already exists
	found := &appsv1.StatefulSet{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new StatefulSet", "Namespace", obj.Namespace, "Name", obj.Name)
		err = r.Client.Create(context.TODO(), obj)
		if err != nil {
			return false, err
		}
		// Pod created successfully - don't requeue
		return false, nil
	} else if err != nil {
		return false, err
	}

	r.Log.Info("Check spec of StatefulSet", "Namespace", obj.Namespace, "Name", obj.Name)
	if !reflect.DeepEqual(found.Spec.Template.Spec, obj.Spec.Template.Spec) {
		r.Log.Info("Difference between Current and Desired state has been found")

		obj.ResourceVersion = found.GetResourceVersion()
		err = r.Client.Update(context.TODO(), obj)
		if err != nil {
			r.Log.Error(err, "Cannot update number of replicas for StatefulSet")
			return true, err
		}
	} else {
		r.Log.Info("StatefulSet looks updated")
	}

	// Pod already exists - don't requeue
	r.Log.Info("Skip reconcile: StatefulSet already exists", "Namespace", found.Namespace, "Name", found.Name)
	return false, nil
}

func (r *KafkaClusterReconciler) deleteOldKafkaStatefulSet(labelString string) error {
	stsList := &appsv1.StatefulSetList{}
	labelSelector := labels.NewSelector()

	req, err := labels.ParseToRequirements(labelString)
	if err != nil && errors.IsNotFound(err) {
		msg := fmt.Sprintf("Cannot parse label %v to requirement", labelString)
		r.Log.Error(err, msg)
		return err
	}

	for _, r := range req {
		labelSelector = labelSelector.Add(r)
	}

	listOptions := &client.ListOptions{}
	listOptions.LabelSelector = labelSelector
	listOptions.Namespace = r.kafka.GetNamespace()

	err = r.Client.List(context.TODO(), stsList, listOptions)
	if err != nil && errors.IsNotFound(err) {
		msg := fmt.Sprintf("Cannot get K8s stsList with labels %v", labelString)
		r.Log.Error(err, msg)
		return err
	}
	for _, sts := range stsList.Items {
		err = r.Client.Delete(context.TODO(), sts.DeepCopy())
		if err != nil && !errors.IsNotFound(err) {
			r.Log.Error(err, "Deleting ", sts.Name)
			return err
		} else if errors.IsNotFound(err) {
			return nil
		}
		r.Log.Info("Old version of inconsistent StatefulSet has been deleted", "StatefulSet", sts.Name)
	}
	return nil
}

func (r *KafkaClusterReconciler) handleSVCsKafka() (bool, error) {
	// Define a new object
	objHeadless := getKafkaServiceHeadless(r.kafka)

	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, objHeadless, r.Scheme); err != nil {
		return false, err
	}

	// Check if this Service already exists
	foundHeadless := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: objHeadless.Name, Namespace: objHeadless.Namespace}, foundHeadless)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Service", "Namespace", objHeadless.Namespace, "Name", objHeadless.Name)
		err = r.Client.Create(context.TODO(), objHeadless)
		if err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	} else {
		// Pod already exists - continue to next svc
		r.Log.Info("Skip reconcile: Service already exists", "Namespace", foundHeadless.Namespace, "Name", foundHeadless.Name)
	}

	// Define a new object
	obj := getKafkaService(r.kafka)
	// Set KafkaCluster instance as the owner and controller
	if err := controllerutil.SetControllerReference(r.kafka, obj, r.Scheme); err != nil {
		return false, err
	}
	// Check if this Service already exists
	found := &corev1.Service{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new Service", "Namespace", obj.Namespace, "Name", obj.Name)
		err = r.Client.Create(context.TODO(), obj)
		if err != nil {
			return false, err
		}

		// Pod created successfully - don't requeue
		return false, nil
	} else if err != nil {
		return false, err
	}
	r.Log.Info("Skip reconcile: Service already exists", "Namespace", found.Namespace, "Name", found.Name)

	if len(r.kafka.Spec.Options.JMXExporterImage) > 0 {
		// Define a new object
		obj := getKafkaExporterService(r.kafka)
		// Set KafkaCluster instance as the owner and controller
		if err := controllerutil.SetControllerReference(r.kafka, obj, r.Scheme); err != nil {
			return false, err
		}
		// Check if this Service already exists
		found := &corev1.Service{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			r.Log.Info("Creating a new Service", "Namespace", obj.Namespace, "Name", obj.Name)
			err = r.Client.Create(context.TODO(), obj)
			if err != nil {
				return false, err
			}

			// Pod created successfully - don't requeue
			return false, nil
		} else if err != nil {
			return false, err
		}
		r.Log.Info("Skip reconcile: Service already exists", "Namespace", found.Namespace, "Name", found.Name)

	}

	return false, nil
}

func (r *KafkaClusterReconciler) UpdateClusterStatus() (bool, error) {
	// Define a new object
	obj := getKafkaStatefulSet(r.kafka)

	sts := &appsv1.StatefulSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}, sts)
	if err != nil {
		return false, err
	}

	if *sts.Spec.Replicas == sts.Status.ReadyReplicas {
		r.kafka.Status.ClusterStatus = "Done"
	} else {
		r.kafka.Status.ClusterStatus = fmt.Sprintf("Starting,  %d unready members", (*sts.Spec.Replicas - sts.Status.ReadyReplicas))
	}
	r.kafka.Status.ReadyMembers = sts.Status.ReadyReplicas

	// Update CR status
	err = r.Client.Update(context.TODO(), r.kafka)
	if err != nil {
		return true, err
	}

	return false, nil
}

func (r *KafkaClusterReconciler) handleConfigMap() (bool, error) {
	cfg := getConfigMapJXMKafkaExporter(r.kafka)

	foundCfg := &corev1.ConfigMap{}

	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: cfg.Name, Namespace: cfg.Namespace}, foundCfg)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info(fmt.Sprintf("Creating a new Config Map Name %s", cfg.Name))
		err = r.Client.Create(context.TODO(), cfg)
		if err != nil {
			return false, err
		}
	} else {
		r.Log.Info(fmt.Sprintf("Skip reconcile: ConfigMap already exists Name %s", foundCfg.Name))
		if !reflect.DeepEqual(cfg.Data, foundCfg.Name) {
			foundCfg.Data = cfg.Data
			r.Log.Info(fmt.Sprintf("Updating ConfigMap %s", foundCfg.Name))

			err = r.Client.Update(context.TODO(), foundCfg)
			if err != nil {
				r.Log.Error(err, fmt.Sprintf("Cannot update ConfigMap %s", foundCfg.Name))
				return false, err
			}
		}
	}

	return false, nil
}
