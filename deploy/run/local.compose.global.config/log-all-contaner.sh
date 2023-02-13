docker logs app-db 1>./log/app-db.log 2>&1
docker logs app-api 1>./log/app-api.log 2>&1
docker logs app-ui 1>./log/app-ui.log 2>&1
docker logs docker-app-api 1>./log/docker-app-api.log 2>&1
docker logs app-liquibase 1>./log/app-liquibase.log 2>&1

