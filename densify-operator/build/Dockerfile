FROM quay.io/operator-framework/helm-operator:v0.19.2

LABEL name="Densify Operator" \
      vendor="Densify" \
      maintainer="support@densify.com" \
      version="v0.0.1" \
      release="1" \
      summary="Densify Operator to deploy Densify Container Data Collection for Kubernetes" \
      description="This Operator will deploy the Densify Container Data Collection Forwarder to the cluster"

### add licenses to this directory
COPY ./LICENSE /licenses/LICENSE

COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts/ ${HOME}/helm-charts/
