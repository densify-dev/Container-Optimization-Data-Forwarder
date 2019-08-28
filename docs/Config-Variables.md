The following table briefly explains the default variable values, environment and command line variable names.

THe order of precedence is Command Line, Config File, Environment Variables. 

## Variable Names
| Config Setting Name | Default | Environment Variables | Config.Properties | Command Line |
|--------|-------|-------|-------|-------|
| Cluster Name | "" | PROMETHEUS_CLUSTER | cluster_name | clusterName | 
| Prometheus Protocol | http | PROMETHEUS_PROTOCOL | protocol | protocol |
| Prometheus Address | "" | PROMETHEUS_ADDRESS | prometheus_address | address | 
| Prometheus Port | 9090 | PROMETHEUS_PORT | prometheus_port | port |
| Interval | hours | PROMETHEUS_INTERVAL | interval | interval |
| Interval Size | 1 | PROMETHEUS_INTERVALSIZE | interval_size | intervalSize |
| History | 1 | PROMETHEUS_HISTORY | history | history | 
| Offset | 0 | PROMETHEUS_OFFSET | offset | offset | 
| Debug | false | PROMETHEUS_DEBUG | debug | debug |
| Config File | config | PROMETHEUS_CONFIGFILE | N/A | file |
| Config Path | ./config | PROMETHEUS_CONFIGPATH | N/A | path |


