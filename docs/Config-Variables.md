The following table briefly explains the default variable values, environment and command line variable names.

NOTE: IF THE CONFIG FILE IS FILLED OUT AND YOU PASS COMMAND LINE 
ARGUMENTS, IT WILL DEFAULT TO THE CONFIG FILE VARIABLES

## Variable Names
| Name | Default | Environment Variable | Command Line Variables |
|--------|-------|-------|-------|
| Cluster Name | "" | PROMETHEUS_CLUSTER | --clusterName="INPUT" | 
| Prometheus Protocol | http | PROMETHEUS_PROTOCOL | --protocol="INPUT" |
| Prometheus Address | "" | PROMETHEUS_ADDRESS | --address="INPUT" | 
| Prometheus Port | 9090 | PROMETHEUS_PORT | --port="INPUT" |
| Interval | hours | PROMETHEUS_INTERVAL | --interval="INPUT" |
| Interval Size | 1 | PROMETHEUS_INTERVALSIZE | --intervalSize="INPUT" |
| History | 1 | PROMETHEUS_HISTORY | --history="INPUT" | 
| Offset | 0 | PROMETHEUS_OFFSET | --offset="INPUT" | 
| Debug | false | PROMETHEUS_DEBUG | --debug="INPUT" |
| Config File | config | PROMETHEUS_CONFIGFILE | --file="INPUT" |
| Config Path | ./config | PROMETHEUS_CONFIGPATH | --path="INPUT" |


