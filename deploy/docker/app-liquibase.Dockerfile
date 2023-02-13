##
## Deploy stage
##
FROM liquibase/liquibase:4.9.1

COPY ./db/liquibase/changelog/. /liquibase/changelog/
#COPY ./db/liquibase/liquibase-compose.properties /liquibase/properties/
COPY ./db/sql /sql
