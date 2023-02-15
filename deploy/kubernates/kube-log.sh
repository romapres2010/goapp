if [ $# -eq 0 ]
then
    echo "No arguments supplied"
    exit 1
fi

if [ -z "$1" ]
then
    echo "No kube variant supplied"
    exit 1
fi

if [ -z "$2" ]
then
    echo "No namespace supplied"
    exit 1
fi

export KUBE_NAMESPACE=$2
echo "Kube namespace:" $KUBE_NAMESPACE

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

export APP_DB_LOG_FILE=$(pwd)/log/log-$(echo $KUBE_VARIANT)-app-db.log
echo "APP db Log file:" $APP_DB_LOG_FILE

kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-db,variant=$KUBE_VARIANT 1>$APP_DB_LOG_FILE 2>&1

export APP_API_LOG_FILE=$(pwd)/log/log-$(echo $KUBE_VARIANT)-app-api.log
echo "APP api Log file:" $APP_API_LOG_FILE

kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-api,variant=$KUBE_VARIANT 1>$APP_API_LOG_FILE 2>&1

export APP_LIQUIBASE_LOG_FILE=$(pwd)/log/log-$(echo $KUBE_VARIANT)-app-liquibase.log
echo "APP liquibase Log file:" $APP_LIQUIBASE_LOG_FILE

kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-liquibase,variant=$KUBE_VARIANT 1>$APP_LIQUIBASE_LOG_FILE 2>&1

