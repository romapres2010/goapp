export ENV_FILE=$(pwd)/.env
echo "Environment file:" $ENV_FILE
export LOG_FILE=$(pwd)/log/compose-app-down.log
echo "Log file:" $LOG_FILE
(cd .. && ./_script/compose-app-down.sh $ENV_FILE 1>$LOG_FILE 2>&1)