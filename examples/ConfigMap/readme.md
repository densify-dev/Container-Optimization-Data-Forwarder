In this example you will find a configmap.yml and pod.yml which will show how you can create a pod, then pass the Config Map to provide the settings needed in the config.cfg.
1. Modify the configmap.yml to point to your Prometheus and Densify Servers.
2. Create the config map in Kubernetes.
    kubectl create -f configmap.yml
3. Create the pod using the pod.yml 
    kubectl create -f pod.yml

The pod will run and send data. Once it has done this it will end so you will need to schedule it to run based on an interval you using for the data collection in the config.
