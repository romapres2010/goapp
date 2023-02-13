export ENV_FILE=$(pwd)/.env
echo "Environment file:" $ENV_FILE
export LOG_FILE=$(pwd)/log/compose-app-hazelcast-up.log
echo "Log file:" $LOG_FILE
(cd .. && ./_script/compose-app-up.sh $ENV_FILE hazelcast  hazelcast-management-center 1>$LOG_FILE 2>&1)
