Configuration 

1. Download a copy of the config.cfg file
2. Modify the config.cfg file to point to your Densify instance and Prometheus server.
3. Run the container providing the updated config.cfg in the /config directory. You can use a Config Map or volume mount for example. Please see [examples](../examples) for sample steps.
4. Schedule the container to run daily or hourly based on your data collection interval selected in the config. 