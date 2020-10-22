# Helm Chart

To deploy it via Helm follow these steps:
1. Clone or update repo
2. Set the relevant endpoints and credentials values in helm/values.yaml (see the [configuration table](Helm-Parameters.md))
3. cd helm
4. Run the command: 'helm install . -f values.yaml'

# Configuration 

1. Download a copy of the config.properties file.
2. Modify the config.properties file to point to your Densify instance and your Prometheus server.
3. Run the container using the updated config.properties in the /config directory. You can use a Config Map or a volume mount, for example. See [examples](../examples) for the sample steps.
4. Schedule the container to run daily or hourly, based on the data collection interval you defined in the config.proerties file. 