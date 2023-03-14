#https://kubernetes.io/ru/docs/reference/kubectl/cheatsheet/

if [ $# -eq 3 ]
then
    echo "Got 3 arguments: [KUBE_VARIANT:$1], [KUBE_NAMESPACE:$2], [LOG_DIR:$3]"
else 
    echo "No enough arguments supplied"
    exit 1
fi

export KUBE_NAMESPACE=$2
echo "Kube namespace:" $KUBE_NAMESPACE

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

export LOG_DIR=$3
echo "Log directory:" $LOG_DIR

export APP_DB_LOG_FILE=$(echo $LOG_DIR)/log-$(echo $KUBE_VARIANT)-app-db.log
echo "APP db Log file:" $APP_DB_LOG_FILE
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-db,variant=$KUBE_VARIANT 1>$APP_DB_LOG_FILE 2>&1

export APP_API_LOG_FILE=$(echo $LOG_DIR)/log-$(echo $KUBE_VARIANT)-app-api.log
echo "APP api Log file:" $APP_API_LOG_FILE
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-api,variant=$KUBE_VARIANT 1>$APP_API_LOG_FILE 2>&1

export APP_LIQUIBASE_LOG_FILE=$(echo $LOG_DIR)/log-$(echo $KUBE_VARIANT)-app-liquibase.log
echo "APP liquibase Log file:" $APP_LIQUIBASE_LOG_FILE

echo "================= APP liquibase =================" 1>>$APP_LIQUIBASE_LOG_FILE 2>&1
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-liquibase,variant=$KUBE_VARIANT 1>>$APP_LIQUIBASE_LOG_FILE 2>&1

echo "================= APP liquibase Install Empty =================" 1>>$APP_LIQUIBASE_LOG_FILE 2>&1
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-liquibase-install-empty,variant=$KUBE_VARIANT 1>>$APP_LIQUIBASE_LOG_FILE 2>&1

echo "================= APP liquibase Install test data =================" 1>>$APP_LIQUIBASE_LOG_FILE 2>&1
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-liquibase-install-testdata,variant=$KUBE_VARIANT 1>>$APP_LIQUIBASE_LOG_FILE 2>&1

echo "================= APP liquibase Upgrade =================" 1>>$APP_LIQUIBASE_LOG_FILE 2>&1
kubectl logs --tail=-1 -n $KUBE_NAMESPACE -l tier=app-liquibase-upgrade,variant=$KUBE_VARIANT 1>>$APP_LIQUIBASE_LOG_FILE 2>&1

