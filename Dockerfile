FROM openjdk:8u191-jdk-alpine3.9
RUN apk add --update python3 py-pip \
 && pip install requests \
 && rm -rf /var/cache/apk/*
COPY ./densify .
CMD ["java", "-jar", "IngestionClient.jar", "-c", "-n", "k8s_transfer_v2", "-l", "k8s_transfer_v2", "-o", "upload", "-r", "-C", "config"]
