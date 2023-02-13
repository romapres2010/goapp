SET APP_CONFIG_FILE=.\..\..\..\..\deploy\config\app.global.yaml
SET APP_HTTP_LISTEN_SPEC=127.0.0.1:3001
SET APP_LOG_LEVEL=INFO
SET APP_LOG_FILE=.\log\app.log
SET APP_PG_USER=postgres
SET APP_PG_PASS=postgres
SET APP_PG_HOST=localhost
SET APP_PG_PORT=5437
SET APP_PG_DBNAME=postgres

.\app.exe

pause