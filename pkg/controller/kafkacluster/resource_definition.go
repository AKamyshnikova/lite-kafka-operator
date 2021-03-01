package kafkacluster

import (
	"fmt"
	"strconv"

	"sigs.k8s.io/yaml"

	litekafkav1alpha1 "github.com/Svimba/lite-kafka-operator/pkg/apis/litekafka/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getKafkaStatefulSet(kafka *litekafkav1alpha1.KafkaCluster) *appsv1.StatefulSet {
	labels := kafka.GetDefaultLabels()
	metaData := metav1.ObjectMeta{
		Namespace: kafka.GetNamespace(),
		Name:      kafka.GetName(),
		Labels:    labels,
	}
	selectors := &metav1.LabelSelector{
		MatchLabels: labels,
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
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		SuccessThreshold:    1,
		FailureThreshold:    3,
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
				StorageClassName: &kafka.Spec.DataStorageClass,
			},
		},
	}
	envVars := []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath:  "status.podIP",
					APIVersion: "v1",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath:  "metadata.name",
					APIVersion: "v1",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath:  "metadata.namespace",
					APIVersion: "v1",
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
		{
			Name:  "JMX_EXPORTER_PORT",
			Value: strconv.FormatUint(uint64(kafka.Spec.Options.JXMExporterPort), 10),
		},
		{
			Name:  "KAFKA_NUM_PARTITIONS",
			Value: "30",
		},
	}
	var cmd []string
	if kafka.Spec.Options.UseExternalAddress {
		cmd = []string{
			`sh`,
			`-exc`,
			`unset KAFKA_PORT && export KAFKA_BROKER_ID=${POD_NAME##*-} && export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://${POD_NAME}-external:` + fmt.Sprintf("%d", kafka.Spec.ContainerPort.Port) + ` && exec /etc/confluent/docker/run`,
		}
	} else {
		cmd = []string{
			`sh`,
			`-exc`,
			`unset KAFKA_PORT && export KAFKA_BROKER_ID=${POD_NAME##*-} && export KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://${POD_IP}:` + fmt.Sprintf("%d", kafka.Spec.ContainerPort.Port) + ` && exec /etc/confluent/docker/run`,
		}
	}
	kafkaContainer := corev1.Container{
		Name:            "kafka-broker",
		Image:           kafka.Spec.Image,
		ImagePullPolicy: corev1.PullAlways,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		Ports: []corev1.ContainerPort{
			{
				Name:          kafka.Spec.ContainerPort.Name,
				ContainerPort: kafka.Spec.ContainerPort.Port,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env:     envVars,
		Command: cmd,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "datadir",
				MountPath: "/opt/kafka/data",
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}
	exporterContainer := corev1.Container{
		Name:            "jmx-exporter",
		Image:           kafka.Spec.Options.JMXExporterImage,
		ImagePullPolicy: corev1.PullAlways,
		LivenessProbe:   &corev1.Probe{Handler: corev1.Handler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("metrics")}}},
		ReadinessProbe:  &corev1.Probe{Handler: corev1.Handler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("metrics")}}},
		Ports: []corev1.ContainerPort{
			{
				Name:          "metrics",
				ContainerPort: int32(kafka.Spec.Options.JXMExporterPort),
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Env: envVars,
		Command: []string{
			`/bin/bash`, `-c`, `java -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap -XX:MaxRAMFraction=1 -XshowSettings:vm -jar jmx_prometheus_httpserver.jar ${JMX_EXPORTER_PORT} /etc/jmx/jmx-prometheus.yml`,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "jmx-config",
				MountPath: "/etc/jmx",
			},
		},
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
	}
	podContainers := []corev1.Container{}
	volumes := []corev1.Volume{}
	podContainers = append(podContainers, kafkaContainer)
	if len(kafka.Spec.Options.JMXExporterImage) > 0 {
		podContainers = append(podContainers, exporterContainer)
		volumes = append(volumes,
			corev1.Volume{
				Name: "jmx-config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "tf-jmx-kafka-config",
						},
					},
				},
			})
	}

	sts := appsv1.StatefulSet{
		ObjectMeta: metaData,
		Spec: appsv1.StatefulSetSpec{
			Replicas:             &replicas,
			Selector:             selectors,
			PodManagementPolicy:  "OrderedReady",
			ServiceName:          kafka.Name + "-headless",
			VolumeClaimTemplates: volumeClaimTemplate,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metaData,
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
					Containers:                    podContainers,
					Volumes:                       volumes,
					Affinity:                      kafka.Spec.Affinity,
					RestartPolicy:                 corev1.RestartPolicyAlways,
				},
			},
		},
	}

	if kafka.Spec.Resources != nil {
		sts.Spec.Template.Spec.Containers[0].Resources = *kafka.Spec.Resources
	}
	return &sts
}

func getKafkaServiceHeadless(kafka *litekafkav1alpha1.KafkaCluster) *corev1.Service {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-kafka-headless",
		Labels:    kafka.GetDefaultLabels(),
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
			Selector:  kafka.GetDefaultLabels(),
		},
	}

	return &service
}

func getKafkaService(kafka *litekafkav1alpha1.KafkaCluster) *corev1.Service {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-kafka",
		Labels:    kafka.GetDefaultLabels(),
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
			Selector: kafka.GetDefaultLabels(),
		},
	}

	return &service
}

func getKafkaExporterService(kafka *litekafkav1alpha1.KafkaCluster) *corev1.Service {
	metaData := metav1.ObjectMeta{
		Namespace: kafka.Namespace,
		Name:      kafka.Name + "-exporter",
		Labels:    kafka.GetDefaultLabels(),
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
		},
	}

	service := corev1.Service{
		ObjectMeta: metaData,
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http-metrics",
					Port:       int32(kafka.Spec.Options.JXMExporterPort),
					TargetPort: intstr.IntOrString{IntVal: int32(kafka.Spec.Options.JXMExporterPort)},
				},
			},
			Selector: kafka.GetDefaultLabels(),
		},
	}

	return &service
}

func getConfigMapJXMKafkaExporter(kafka *litekafkav1alpha1.KafkaCluster) *corev1.ConfigMap {
	conf, err := yaml.Marshal(kafka.Spec.Options.JMXExporterRules)
	if err != nil {
		log.Error(err, "Cannot load Kafka JMX exporter config yaml")
		conf = []byte("")
	}
	jmxConf := string(conf)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tf-jmx-kafka-config",
			Namespace: kafka.Namespace,
		},
		Data: map[string]string{
			"jmx-prometheus.yml": jmxConf,
		},
	}
	return cm
}
