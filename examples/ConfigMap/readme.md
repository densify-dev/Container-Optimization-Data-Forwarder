In this example you will find a configmap.yml and pod.yml which will show how can create a pod passing it the Config Map to provide the settings needed in the config.cfg.
1. Modify the configmap.yml to point to your Prometheus and Densify Servers
2. Create the config map in Kubernetes 
kubectl create -f configmap.yml
3. Create the pod using the pod.yml 
kubectl create -f pod.yml

The pod will run and send data over once it has done this it will end so you will need to schedule it to run based on interval you using for the data collection in the config.