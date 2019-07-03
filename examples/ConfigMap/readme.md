In this example you will edit the configmap.yml and create the pod. You will then pass the Config Map, containing the required settings, to the config.properties file.
1. Modify the configmap.yml to point to your Densify instance and to your Prometheus server.
2. Create the config map in Kubernetes:
    ```kubectl create -f configmap.yml```
3. Create the pod using pod.yml: 
    ```kubectl create -f pod.yml```

The pod will run and send the collected data to Densify. Once that data has been sent the pod will end. You will need to schedule the pod to run on the same interval you are using for data collection, as defined in the config file.
