##
## Deploy stage
##
FROM liquibase/liquibase:4.9.1

# Changelog для различных сред DEV-TEST-PROD можно встроить в сборку и переключаться через ENV переменные
COPY ./db/liquibase/changelog/. /liquibase/changelog/

# DDL и DML скрипты можно встроить в сборку
COPY ./db/sql /sql
