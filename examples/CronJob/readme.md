In this example you will find a configmap.yml and cronjob.yml that shows you how to create a cron job that is scheduled to run every hour, then pass the Config Map that provides the settings required in the config.properties.
1. Modify the configmap.yml to point to your Prometheus and Densify Servers.
2. Create the config map in Kubernetes.
    kubectl create -f configmap.yml
3. Create the cron job using the cronjob.yml 
    kubectl create -f cronjob.yml

The cron job will run and send data to Densify once an hour. Once that data has been sent the job will end. You should adjust the schedule of the cron job to run on the interval you are using for data collection, as defined in the config file.
