The following tables provide the default values and the environment and command line names for the variables used to configure the data forwarder.

The order of precedence is: Command Line, Config File, Environment Variables. 

## Variable Names Data Collection
| Config Setting Name | Default Value | Environment Variables | Config.Properties | Command Line |
|--------|-------|-------|-------|-------|
| Cluster Name | "" | PROMETHEUS_CLUSTER | cluster_name | clusterName | 
| Prometheus Protocol | http | PROMETHEUS_PROTOCOL | protocol | protocol |
| Prometheus Address | "" | PROMETHEUS_ADDRESS | prometheus_address | address | 
| Prometheus Port | 9090 | PROMETHEUS_PORT | prometheus_port | port |
| Interval | hours | PROMETHEUS_INTERVAL | interval | interval |
| Interval Size | 1 | PROMETHEUS_INTERVALSIZE | interval_size | intervalSize |
| History | 1 | PROMETHEUS_HISTORY | history | history | 
| Sample Rate | 5 | PROMETHEUS_SAMPLERATE | sample_rate | sampleRate |
| Offset | 0 | PROMETHEUS_OFFSET | offset | offset | 
| Include List | container,node,nodegroup,cluster,quota | PROMETHEUS_INCLUDE | include_list | includeList |
| Node Group List | label_cloud_google_com_gke_nodepool,label_eks_amazonaws_com_nodegroup,label_agentpool,label_pool_name,label_alpha_eksctl_io_nodegroup_name,label_kops_k8s_io_instancegroup | NODE_GROUP_LIST | node_group_list | nodeGroupList |
| Debug | false | PROMETHEUS_DEBUG | debug | debug |
| Config File | config | PROMETHEUS_CONFIGFILE | N/A | file |
| Config Path | ./config | PROMETHEUS_CONFIGPATH | N/A | path |
| OAuth Token | "" | OAUTH_TOKEN | prometheus_oauth_token | oAuthToken |
| CA Certificate| "" | CA_CERT | ca_certificate | caCert |

## Variable Names Forwarder
| Config Setting Name  | Environment Variable | 
|--------|-------|
| Host | DENSIFY_HOST |
| Protocol | DENSIFY_PROTOCOL |
| Port | DENSIFY_PORT |
| Endpoint | DENSIFY_ENDPOINT |
| User | DENSIFY_USER |
| Proxy Host | DENSIFY_PROXYHOST | 
| Proxy Port | DENSIFY_PROXYPORT |
| Proxy Protocol | DENSIFY_PROXYPROTOCOL | 
| Proxy Auth | DENSIFY_PROXYAUTH |
| Proxy User | DENSIFY_PROXYUSER |
| Proxy Password | DENSIFY_PROXYPASSWORD | 
| Encrypted Proxy Password | DENSIFY_EPROXYPASSWORD | 
| Proxy Server | DENSIFY_PROXYSERVER |
| Proxy Domain | DENISFY_PROXYDOMAIN | 
| Debug | DENSIFY_DEBUG | 
