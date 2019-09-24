# Helm chart Configuration

The following table lists the configurable parameters of the container-optimization-data-forwarder chart in helm/resources/overrideValues.yaml.

| Parameter                                | Description                                             | Default                   |
|------------------------------------------|---------------------------------------------------------|---------------------------|
| `config.densify.hostname`        | Host Name / IP of the Densify server                            |                 |
| `config.densify.port`            | Port of the Densify server                                      |                 |
| `config.densify.protocol`        | Protocol for Densify server connectivity (http/https)           |                 |
| `config.densify.user`            | Username to access Densify server                               |                 |
| `config.densify.password`        | Password to access Densify server                               |                 |
| `config.densify.epassword`       | Encrypted password for Densify server                             |                 |
| `config.prometheus.hostname`     | Host Name / IP of the Prometheus server                         |                 |
| `config.prometheus.port`         | Port to connect in Prometheus server                            |                 |
| `config.prometheus.clustername`  | Prometheus cluster name (optional)                              |                 |
| `config.zipEnabled`              | Controls whether contents are zipped before transmission        | true            |
| `config.zipname`                 | Name of the zip file that archives the content                  |                 |
| `config.proxy.host`              | Host Name of Proxy server                                   |                 |
| `config.proxy.port`              | Port of Proxy server                                        |                 |
| `config.proxy.protocol`          | Protocol of Proxy server (http/https)                       |                 |
| `config.proxy.auth`              | Authentication type of Proxy server  (Basic/NTLM)            |                 |
| `config.proxy.user`              | User Name of Proxy server                                   |                 |
| `config.proxy.password`          | Password of Proxy server                                    |                 |
| `config.proxy.epassword`         | Encrypted password for Proxy server                                   |                 |
| `config.proxy.domainuser`        | Domain username (NTLM authentication)                           |                 |
| `config.proxy.domain`            | Domain name (NTLM authentication)                               |                 |
| `config.debug`                   | Enable debugging                                                | false           |
| `config.debugkey`                | Debug key                                                       |                 |
