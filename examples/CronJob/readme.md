In this example you will edit the configmap.yml and create a cron job to run data collection every hour and pass the Config Map containing the required settings, to the config.properties file.
1. Modify the configmap.yml to point to your Densify instance and Prometheus server.
2. Create the config map in Kubernetes.
    
    `kubectl create -f configmap.yml`
3. Create the cron job using the cronjob.yml 
    
    `kubectl create -f cronjob.yml`

The cron job will run and send data to Densify hourly. You should adjust the cron job schedule to run on the same interval you are using for data collection, as defined in the config.properties file.
