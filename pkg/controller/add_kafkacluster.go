package controller

import (
	"github.com/Svimba/lite-kafka-operator/pkg/controller/kafkacluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kafkacluster.Add)
}
