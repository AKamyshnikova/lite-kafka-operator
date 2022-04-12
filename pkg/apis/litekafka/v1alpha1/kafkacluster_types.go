package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Port defines the desired state of Port
// +k8s:openapi-gen=true
type Port struct {
	Name string `json:"name"`
	Port int32  `json:"port"`
}

// ZookeeperSpec defines the desired state of ZookeeperSpec
// +k8s:openapi-gen=true
type ZookeeperSpec struct {
	Host string `json:"host"`
	Port *Port  `json:"port"`
}

type Rule struct {
	Name    string            `json:"name,omitempty"`
	Pattern string            `json:"pattern"`
	Type    string            `json:"type,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type JMXExporterConfig struct {
	JmxURL                    string   `json:"jmxUrl"`
	LowercaseOutputLabelNames *bool    `json:"lowercaseOutputLabelNames"`
	LowercaseOutputName       *bool    `json:"lowercaseOutputName"`
	Rules                     []Rule   `json:"rules"`
	Ssl                       *bool    `json:"ssl"`
	WhitelistObjectNames      []string `json:"whitelistObjectNames"`
}

// KafkaOptions defines the desired state of KafkaOptions
// +k8s:openapi-gen=true
type KafkaOptions struct {
	TopicReplicationFactor uint               `json:"topicReplicationFactor,omitempty"`
	JXMPort                uint               `json:"jxmPort,omitempty"`
	JXMExporterPort        uint               `json:"jxmExporterPort,omitempty"`
	JMXExporterImage       string             `json:"jmxExporterImage,omitempty"`
	JMXExporterRules       *JMXExporterConfig `json:"jmxExporterRules,omitempty"`
	UseExternalAddress     bool               `json:"use_external_address,omitempty"`
}

// KafkaClusterSpec defines the desired state of KafkaCluster
// +k8s:openapi-gen=true
type KafkaClusterSpec struct {
	Replicas       int32          `json:"replicas"`
	ContainerPort  *Port          `json:"containerPort"`
	ServicePort    *Port          `json:"servicePort"`
	Storage        string         `json:"storage"`
	Options        *KafkaOptions  `json:"options,omitempty"`
	Zookeeper      *ZookeeperSpec `json:"zookeeper"`
	ZookeeperCheck *bool          `json:"zookeeperCheck"`
	Image          string         `json:"image"`
	// +kubebuilder:default:=Always
	ImagePullPolicy  corev1.PullPolicy            `json:"imagePullPolicy,omitempty"`
	DataStorageClass string                       `json:"dataStorageClass,omitempty"`
	Affinity         *corev1.Affinity             `json:"affinity,omitempty"`
	Resources        *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// KafkaClusterStatus defines the observed state of KafkaCluster
// +k8s:openapi-gen=true
type KafkaClusterStatus struct {
	ClusterStatus string `json:"clusterStatus"`
	ReadyMembers  int32  `json:"readyMembers"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaCluster is the Schema for the kafkaclusters API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KafkaCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaClusterSpec   `json:"spec,omitempty"`
	Status KafkaClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KafkaClusterList contains a list of KafkaCluster
type KafkaClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KafkaCluster{}, &KafkaClusterList{})
}

func (kc *KafkaCluster) GetDefaultLabels() map[string]string {
	return map[string]string{
		"app":       kc.GetName(),
		"component": "kafka-broker",
		"name":      "kafka",
	}
}

func (conf *JMXExporterConfig) setDefaultJMXConf() {
	defaultFalse := false
	defaultTrue := true
	if len(conf.JmxURL) == 0 {
		conf.JmxURL = "service:jmx:rmi:///jndi/rmi://127.0.0.1:5555/jmxrmi"
	}
	if conf.Ssl == nil {
		conf.Ssl = &defaultFalse
	}
	if conf.LowercaseOutputLabelNames == nil {
		conf.LowercaseOutputLabelNames = &defaultTrue
	}
	if conf.LowercaseOutputName == nil {
		conf.LowercaseOutputName = &defaultTrue
	}
	if len(conf.WhitelistObjectNames) == 0 {
		conf.WhitelistObjectNames = []string{
			"kafka.controller:*",
			"kafka.server:*",
			"java.lang:*",
			"kafka.network:*",
			"kafka.log:*",
		}
	}
	if len(conf.Rules) == 0 {
		ruleName := "kafka_$1_$2_$3"
		ruleCountName := "kafka_$1_$2_$3_count"
		typeGauge := "GAUGE"
		typeCounter := "COUNTER"
		conf.Rules = []Rule{
			{
				Pattern: "^(?!.*>(?:[Ss]td[Dd]ev|[Ff]ive[Mm]inute[Rr]ate|[Ff]ifteen[Mm]inute[Rr]ate|[Cc]ount|(\\d+)th[Pp]ercentile|[Mm]in|[Mm]ean$|[Mm]ax|[Oo]ne[Mm]inute[Rr]ate|[Mm]ean[Rr]ate)$).*$",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"stat": "deviation"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Ss]td[Dd]ev",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "stat": "deviation"},
				Pattern: "kafka.(\\\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Ss]td[Dd]ev",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "stat": "deviation"},
				Pattern: "kafka.(\\\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Ss]td[Dd]ev",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"rate": "1m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Oo]ne[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "rate": "1m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Oo]ne[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "rate": "1m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Oo]ne[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"stat": "minimum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Mm]in",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "stat": "minimum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Mm]in",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "stat": "minimum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Mm]in",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"rate": "mean"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Mm]ean[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "rate": "mean"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Mm]ean[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "rate": "mean"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Mm]ean[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"stat": "average"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Mm]ean[^Rr]",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "stat": "average"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Mm]ean[^Rr]",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "stat": "average"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Mm]ean[^Rr]",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"stat": "maximum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Mm]ax",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "stat": "maximum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Mm]ax",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "stat": "maximum"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Mm]ax",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"rate": "5m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Ff]ive[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "rate": "5m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Ff]ive[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "rate": "5m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Ff]ive[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"rate": "15m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Ff]ifteen[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "rate": "15m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>[Ff]ifteen[Mm]inute[Rr]ate",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "rate": "15m"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Ff]ifteen[Mm]inute[Rr]ate",
			},
			{
				Name: ruleCountName, Type: typeCounter,
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>[Cc]ount",
			},
			{
				Name: ruleCountName, Type: typeCounter, Labels: map[string]string{"$4": "$5", "$6": "$7"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>[Cc]ount",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"quantile": "0.$4"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+)><>(\\d+)th[Pp]ercentile",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "quantile": "0.$6"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+)><>(\\d+)th[Pp]ercentile",
			},
			{
				Name: ruleName, Type: typeGauge, Labels: map[string]string{"$4": "$5", "$6": "$7", "quantile": "0.$8"},
				Pattern: "kafka.(\\w+)<type=(.+), name=(.+), (.+)=(.+), (.+)=(.+)><>(\\d+)th[Pp]ercentile",
			},
		}
	}
}

// SetDefaults Set dfault values of KafkaClusterSpec
func (kc *KafkaCluster) SetDefaults() {
	if kc.Spec.Replicas == 0 {
		kc.Spec.Replicas = 3
	}
	if len(kc.Spec.Storage) == 0 {
		kc.Spec.Storage = "1Gi"
	}
	if kc.Spec.ContainerPort == nil {
		kc.Spec.ContainerPort = &Port{Name: "kafka", Port: DefaultKafkaBrokerPort}
	}
	if kc.Spec.ServicePort == nil {
		kc.Spec.ServicePort = &Port{Name: "broker", Port: DefaultKafkaServicePort}
	}
	if kc.Spec.ZookeeperCheck == nil {
		enable := true
		kc.Spec.ZookeeperCheck = &enable
	}
	if kc.Spec.Zookeeper == nil {
		kc.Spec.Zookeeper = &ZookeeperSpec{
			Host: "zookeeper",
			Port: &Port{Name: "zookeeper", Port: DefaultZookeeperPort},
		}
	} else {
		if len(kc.Spec.Zookeeper.Host) == 0 {
			kc.Spec.Zookeeper.Host = "zookeeper"
		}
		if kc.Spec.Zookeeper.Port == nil {
			kc.Spec.Zookeeper.Port = &Port{Name: "zookeeper", Port: DefaultZookeeperPort}
		}
	}
	if kc.Spec.Options == nil {
		kc.Spec.Options = &KafkaOptions{
			TopicReplicationFactor: 2,
			JXMPort:                DefaultKafkaJMXPort,
			JXMExporterPort:        DefaultKafkaExporterJMXPort,
			JMXExporterRules:       &JMXExporterConfig{},
		}
	} else {
		if kc.Spec.Options.TopicReplicationFactor == 0 {
			kc.Spec.Options.TopicReplicationFactor = 2
		}
		if kc.Spec.Options.JXMExporterPort == 0 {
			kc.Spec.Options.JXMExporterPort = DefaultKafkaExporterJMXPort
		}
		if kc.Spec.Options.JXMPort == 0 {
			kc.Spec.Options.JXMPort = DefaultKafkaJMXPort
		}
		if kc.Spec.Options.JMXExporterRules == nil {
			kc.Spec.Options.JMXExporterRules = &JMXExporterConfig{}
		}
	}
	kc.Spec.Options.JMXExporterRules.setDefaultJMXConf()

	if kc.Spec.Affinity == nil {
		kc.Spec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
					{
						Weight: 20,
						PodAffinityTerm: corev1.PodAffinityTerm{
							TopologyKey: "kubernetes.io/hostname",
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{
										Key:      "app",
										Operator: metav1.LabelSelectorOpIn,
										Values:   []string{kc.GetName()},
									},
								},
							},
						},
					},
				},
			},
		}
	}
}
