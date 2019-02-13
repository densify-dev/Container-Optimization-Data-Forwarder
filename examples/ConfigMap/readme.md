In this example you will find a configmap.yml and pod.yml that shows you how to create a pod, then pass the Config Map that provides the settings required in the config.cfg.
1. Modify the configmap.yml to point to your Prometheus and Densify Servers.
2. Create the config map in Kubernetes.
    kubectl create -f configmap.yml
3. Create the pod using the pod.yml 
    kubectl create -f pod.yml

The pod will run and send data. Once that data has been sent the pod will end. You will need to schedule the pod to run on the interval you are using for data collection, as defined in the config file.
