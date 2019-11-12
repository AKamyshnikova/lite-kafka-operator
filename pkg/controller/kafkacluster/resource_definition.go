package kafkacluster

import (
	"fmt"
	"strconv"

	litekafkav1alpha1 "github.com/Svimba/lite-kafka-operator/pkg/apis/litekafka/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getKafkaStatefulSet(kafka *litekafkav1alpha1.KafkaCluster) *appsv1.StatefulSet {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-kafka",
		Labels: map[string]string{
			"app.kubernetes.io/component": "kafka-broker",
			"app.kubernetes.io/name":      "kafka",
			"app.kubernetes.io/instance":  kafka.Name,
		},
	}
	selectors := &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"app.kubernetes.io/component": "kafka-broker",
			"app.kubernetes.io/name":      "kafka",
			"app.kubernetes.io/instance":  kafka.Name,
		},
	}
	replicas := kafka.Spec.Replicas
	terminationGracePeriodSeconds := int64(60)
	livenessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{
				Command: []string{
					"sh",
					"-ec",
					"/usr/bin/jps | /bin/grep -q SupportedKafka",
				},
			},
		},
		InitialDelaySeconds: 30,
		TimeoutSeconds:      5,
	}
	readinessProbe := &corev1.Probe{
		Handler: corev1.Handler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.IntOrString{StrVal: kafka.Spec.ContainerPort.Name, IntVal: kafka.Spec.ContainerPort.Port},
			},
		},
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	volumeClaimTemplate := []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "datadir",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(kafka.Spec.Storage),
					},
				},
			},
		},
	}
	envVars := []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name:  "KAFKA_HEAP_OPTS",
			Value: "-Xmx1G -Xms1G",
		},
		{
			Name:  "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR",
			Value: strconv.FormatUint(uint64(kafka.Spec.Options.TopicReplicationFactor), 10),
		},
		{
			Name:  "KAFKA_ZOOKEEPER_CONNECT",
			Value: fmt.Sprintf("%s:%d", kafka.Spec.Zookeeper.Host, kafka.Spec.Zookeeper.Port.Port),
		},
		{
			Name:  "KAFKA_LOG_DIRS",
			Value: "/opt/kafka/data/logs",
		},
		{
			Name:  "KAFKA_CONFLUENT_SUPPORT_METRICS_ENABLE",
			Value: "false",
		},
		{
			Name:  "KAFKA_JMX_PORT",
			Value: strconv.FormatUint(uint64(kafka.Spec.Options.JXMPort), 10),
		},
	}

	sts := appsv1.StatefulSet{
		ObjectMeta: metaData,
		Spec: appsv1.StatefulSetSpec{
			Replicas:            &replicas,
			Selector:            selectors,
			PodManagementPolicy: "OrderedReady",
			ServiceName:         kafka.Name + "-headless",
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: "OnDelete",
			},
			VolumeClaimTemplates: volumeClaimTemplate,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metaData,
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:            "kafka-broker",
							Image:           kafka.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							LivenessProbe:   livenessProbe,
							ReadinessProbe:  readinessProbe,
							Ports: []corev1.ContainerPort{
								{
									Name:          kafka.Spec.ContainerPort.Name,
									ContainerPort: kafka.Spec.ContainerPort.Port,
								},
							},
							Env: envVars,
							Command: []string{
								`sh`,
								`-exc`,
								`unset KAFKA_PORT && export KAFKA_BROKER_ID=${POD_NAME##*-} && export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://${POD_IP}:` + fmt.Sprintf("%d", kafka.Spec.ContainerPort.Port) + ` && exec /etc/confluent/docker/run`,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "datadir",
									MountPath: "/opt/kafka/data",
								},
							},
						},
					},
				},
			},
		},
	}
	return &sts
}

func getKafkaServiceHeadless(kafka *litekafkav1alpha1.KafkaCluster) *corev1.Service {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-kafka-headless",
		Labels: map[string]string{
			"app.kubernetes.io/component": "kafka-broker",
			"app.kubernetes.io/name":      "kafka",
			"app.kubernetes.io/instance":  kafka.Name,
		},
		Annotations: map[string]string{
			"service.alpha.kubernetes.io/tolerate-unready-endpoints": "true",
		},
	}

	service := corev1.Service{
		ObjectMeta: metaData,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: kafka.Spec.ServicePort.Name,
					Port: kafka.Spec.ServicePort.Port,
				},
			},
			ClusterIP: "None",
			Selector: map[string]string{
				"app.kubernetes.io/component": "kafka-broker",
				"app.kubernetes.io/name":      "kafka",
				"app.kubernetes.io/instance":  kafka.Name,
			},
		},
	}

	return &service
}

func getKafkaService(kafka *litekafkav1alpha1.KafkaCluster) *corev1.Service {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-kafka",
		Labels: map[string]string{
			"app.kubernetes.io/component": "kafka-broker",
			"app.kubernetes.io/name":      "kafka",
			"app.kubernetes.io/instance":  kafka.Name,
		},
	}

	service := corev1.Service{
		ObjectMeta: metaData,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       kafka.Spec.ServicePort.Name,
					Port:       kafka.Spec.ServicePort.Port,
					TargetPort: intstr.IntOrString{StrVal: kafka.Spec.ContainerPort.Name, IntVal: kafka.Spec.ContainerPort.Port},
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/component": "kafka-broker",
				"app.kubernetes.io/name":      "kafka",
				"app.kubernetes.io/instance":  kafka.Name,
			},
		},
	}

	return &service
}
