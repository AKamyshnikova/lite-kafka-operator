FROM gcr.io/distroless/static:nonroot

ENV OPERATOR=/usr/local/bin/kafka-k8s-operator

COPY build/_output/bin/kafka-k8s-operator ${OPERATOR}

ENTRYPOINT ["/usr/local/bin/kafka-k8s-operator"]