Configuration 

1. Download a copy of the config.cfg file.
2. Modify the config.cfg file to point to your Densify instance and your Prometheus server.
3. Run the container using the updated config.cfg in the /config directory. You can use a Config Map or a volume mount, for example. See [examples](../examples) for the sample steps.
4. Schedule the container to run daily or hourly, based on the data collection interval youe selected in the config.cfg file. 
