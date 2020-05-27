NAMESPACE=tf
KUBECTL?=kubectl
KUBECTL_EXTRA?=

##@ Application

kubectl-version: ## Print kunectl version
	- ${KUBECTL} ${KUBECTL_EXTRA} version

install: ## Install all resources (CR/CRD's, RBAC and Operator)
	@echo ....... Creating namespace .......
	- ${KUBECTL} ${KUBECTL_EXTRA} create namespace ${NAMESPACE}
	@echo ....... Applying CRDs .......
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/crds/litekafka_v1alpha1_kafkacluster_crd.yaml
	@echo ....... Applying Rules and Service Account .......
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/role.yaml
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/role_binding.yaml
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/service_account.yaml
	@echo ....... Creating the CRs .......
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/crds/litekafka_v1alpha1_kafkacluster_cr.yaml -n ${NAMESPACE}

run-operator: ## Deploy Operator
	@echo ....... Applying Operator .......
	- ${KUBECTL} ${KUBECTL_EXTRA} apply -f ${PWD}/deploy/operator.yaml -n ${NAMESPACE}

uninstall: ## Uninstall all that all performed in the $ make install (include operator)
	@echo ....... Uninstalling .......
	@echo ....... Deleting CRDs.......
	- ${KUBECTL} ${KUBECTL_EXTRA} delete -f ${PWD}/deploy/crds/litekafka_v1alpha1_kafkacluster_crd.yaml
	@echo ....... Deleting Rules and Service Account .......
	- ${KUBECTL} ${KUBECTL_EXTRA} delete -f ${PWD}/deploy/role.yaml
	- ${KUBECTL} ${KUBECTL_EXTRA} delete -f ${PWD}/deploy/role_binding.yaml
	- ${KUBECTL} ${KUBECTL_EXTRA} delete -f ${PWD}/deploy/service_account.yaml
	@echo ....... Deleting Operator .......
	- ${KUBECTL} ${KUBECTL_EXTRA} delete -f ${PWD}/deploy/operator.yaml -n ${NAMESPACE}
	@echo ....... Deleting namespace ${NAMESPACE}.......
	- ${KUBECTL} ${KUBECTL_EXTRA} delete namespace ${NAMESPACE}


