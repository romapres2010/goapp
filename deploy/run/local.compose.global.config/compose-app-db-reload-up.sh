export ENV_FILE=$(pwd)/.env
echo "Environment file:" $ENV_FILE
export LOG_FILE=$(pwd)/log/compose-app-db-reload-up.log
echo "Log file:" $LOG_FILE
(cd .. && ./_script/compose-app-up.sh $ENV_FILE app-liquibase app-db 1>$LOG_FILE 2>&1)
