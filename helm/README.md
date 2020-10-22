# Densify Container Optimization Helm Chart

<img src="https://www.densify.com/wp-content/uploads/densify.png" width="300">

## Introduction
This chart deploys the Densify Container Optimization Data Forwarder, which collects data from a Prometheus server and sends it to a Densify instance for analysis. 

## Details
* Deploys a configmap, job and cronjob
* The cronjob will run hourly and collect data from Prometheus and send it to Densify for analysis.

## Prerequisites

* Densify account, which is provided with a Densify subscription or through a free trial (https://www.densify.com/service/signup)
* Kubernetes or OpenShift
* Prometheus (https://prometheus.io/)
* Kube-state-metrics version 1.5.0+ (https://github.com/kubernetes/kube-state-metrics)
* Node Exporter (https://hub.docker.com/r/prom/node-exporter) (optional)

## Installing
To deploy it via Helm follow these steps:
1. Clone or update repo
2. Set the relevant endpoints and credentials values in helm/values.yaml (see the [configuration table](Helm-Parameters.md))
3. cd helm
4. Run the command: 
```console
'helm install . -f values.yaml'
```

## Configuration
 
Configure these parameters in values.yaml:

| Parameter        | Description           | default ` |
| ------------- |-------------|--------|
| `nameOverride` | name override for helm chart name. | `densify-forwarder` |
| `image` | Image to use for the Densify Container Optimization Data Forwarder. | `densify/container-optimization-data-forwarder:latest` |
| `pullPolicy` | Image pull policy for the Densify Container Optimization Data Forwarder. | `Always` |
| `config.densify.hostname` | Specify your Densify server host. You may need to specify a fully qualified domain name. | `<instance>.densify.com` |
| `config.densify.protocol` | Specify http or https. | `<http/https>` |
| `config.densify.port` | Specify the Densify connection port. | `443` |
| `config.densify.user` | Specify the Densify user account. This user must already exist in your Densify instance and have API access privileges. | `nil` |
| `config.densify.password` | The password for the Densify user. Only specify one of password or encrypted password. | `nil` |
| `config.densify.epassword` | The encrypted password for the Densify User. Only specify one of passsword or encrypted password. | `nil` |
| `config.densify.UserSecretName` | Name of secret used to store Densify user and epassword. Needs to have keys of username and epassword. Also if using should disable username\password\epassword settings. | `nil` |
| `config.prometheus.hostname` | Specify the Prometheus address. suggested to use the internal service name such as `<service name>.<namespace>.svc`. | `nil` |
| `config.prometheus.protocol` | Specify http or https. | `<http/https>` |
| `config.prometheus.port` | Specify the Prometheus service connection port. | `9090` |
| `config.prometheus.clustername` | Name that will be set for the cluster in Densify UI. If left unset then will use the name that is specified in the prometheus hostname. | `nil` |
| `config.prometheus.interval` | Interval to collect data from Prometheus (hours/days). | `hours` |
| `config.prometheus.intervalSize` | Size of the interval to collect default would be 1 hour based on interval size. | `1` |
| `config.prometheus.history` | History being collected by default it will be just the last hour that is collected. | `1` |
| `config.prometheus.sampleRate` | Sample rate for data points that will be sent to Densify. | `5` |
| `config.prometheus.includeList` | List of included data types being collected. | `container,node,nodegroup,cluster` |
| `config.prometheus.oauth_token` | Oauth token to be used for when need to authenticate to Prometheus is secured environment. | `/var/run/secrets/kubernetes.io/serviceaccount/token` |
| `config.prometheus.ca_certificate` | CA certificate to use when communicating with Prometheus in secure environment. | `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt` |
| `config.proxy.host` | Specify name of the proxy host. | `nil` |
| `config.proxy.port` | Specify the proxy port. | `nil` |
| `config.proxy.protocol` | Specify http or https. | `<http/https>` |
| `config.proxy.auth` | Specify proxy authentication type. | `nil` |
| `config.proxy.user` | Specify the username to use for the proxy. | `nil` |
| `config.proxy.password` | Specify the password to use for the proxy. Should only enable either the proxy password or proxy encrypted password. | `nil` |
| `config.proxy.epassword` | Specify the encrypted password to use for the proxy. Should only enable either the proxy password or proxy encrypted password. | `nil` |
| `config.proxy.domainuser` | Specify domain user for the proxy. | `nil` |
| `config.proxy.domain` | Specify the domain for the proxy. | `nil` |
| `config.zipEnabled` | Should the files be zipped before sending to Densify. | `true` |
| `config.zipname` | Name of the zipfile that will be sent to Densify. | `data/nil` |
| `config.cronJob.schedule` | Schedule to use for the cronjob. Defaults to top of every hour in line with the interval settings of collecting last hour of data. | `0 * * * *` |
| `config.debug` | Turn on debug logging. | `false` |
| `authenticated.create` | Control deployment of service account, cluster role, and cluster role binding for use when the Prometheus server is secured. If using OpenShift environment then likely required to be true. | `false` |
| `nodeSelector` | Node labels for pod assignments. | `{}` |
| `resources` | CPU/Memory resource requests/limits. | `{}` |
| `tolerations` | Toleration lables for pod assignments. | `{}` |

## Limitation
* Supported Architecture: AMD64
* Supported OS: Linux

## Documentation
* [Densify Feature Description and Reference Guide](https://www.densify.com/docs/Content/Welcome.htm)

## License
Apache 2 Licensed. See [LICENSE](LICENSE) for full details.