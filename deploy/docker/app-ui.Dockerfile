##
## Build stage
##

FROM gradle:7.5.1-jdk-alpine AS build

WORKDIR /app

COPY ./ui .

RUN gradle -Pvaadin.productionMode=true bootJar

##
## Run stage
##

FROM openjdk:17.0.2-slim

EXPOSE 8080

RUN mkdir /app

COPY --from=build /app/build/libs/*.jar /app/spring-boot-application.jar

ENTRYPOINT exec java $JAVA_OPTS -jar /app/spring-boot-application.jar
