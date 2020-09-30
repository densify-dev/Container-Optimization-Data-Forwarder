This example shows you how to setup the Data Forwarder to connect to a Prometheus server and send container data to Densify. You need to edit configmap.yml file, then create the config map to pass the settings to config.properties. To test the container data forwarding functionality, create the pod using pod.yml. 

1. Modify the configmap.yml to point to your Densify instance and to your Prometheus server.

2. Create the config map in Kubernetes:
    
    `kubectl create -f configmap.yml`
    
3. Create the pod using pod.yml: 
    
    `kubectl create -f pod.yml`

The pod runs and sends the collected container data to Densify. Once the container data is sent, the pod ends.

You need to schedule the pod to run at the same interval that is configured for data collection, as defined in the config file. See the CronJob example.
