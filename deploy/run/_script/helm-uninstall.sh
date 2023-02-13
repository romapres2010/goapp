#https://kubernetes.io/ru/docs/reference/kubectl/cheatsheet/

if [ $# -eq 4 ]
then
    echo "Got 4 arguments: [HELM_ACTION:$1], [KUBE_VARIANT:$2], [KUBE_NAMESPACE:$3], [LOG_DIR:$4]"
else 
    echo "No enough arguments supplied"
    exit 1
fi

export HELM_ACTION=$1
echo "HELM action:" $HELM_ACTION

export KUBE_VARIANT=$2
echo "Kube variant:" $KUBE_VARIANT

export KUBE_NAMESPACE=$3
echo "Kube namespace:" $KUBE_NAMESPACE

export LOG_DIR=$4
echo "Log directory:" $LOG_DIR

export YAML_FILE=$(echo $LOG_DIR)/helm-$(echo $HELM_ACTION)-$(echo $KUBE_VARIANT)-debug.yaml
echo "YAML file:" $YAML_FILE

export KUBE_LOG_FILE=$(echo $LOG_DIR)/helm-$(echo $HELM_ACTION)-$(echo $KUBE_VARIANT)-kube.log
echo "Kube Log file:" $KUBE_LOG_FILE

export KUBE_DESCRIBE_FILE=$(echo $LOG_DIR)/helm-$(echo $HELM_ACTION)-describe-$(echo $KUBE_VARIANT)-before.log
echo "APP describe file:" $KUBE_DESCRIBE_FILE

export APP_VERSION=$(cat ../version)
export APP_API_APP_NAME=$(cat ../app_api_app_name)
export APP_LUQUIBASE_APP_NAME=$(cat ../app_liquibase_app_name)
export APP_REPOSITORY=$(cat ../default_repository)

echo "================= RUN kube-get.sh ================="
./_script/kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>$KUBE_DESCRIBE_FILE 2>&1

echo "================= RUN kube-descibe.sh ================="
./_script/kube-descibe.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>>$KUBE_DESCRIBE_FILE  2>&1

echo "================= RUN kube-log.sh ================="
./_script/kube-log.sh $KUBE_VARIANT $KUBE_NAMESPACE $LOG_DIR

echo "================= helm list ================="
helm list -n $KUBE_NAMESPACE --all 1>$KUBE_LOG_FILE 2>&1

echo "================= helm uninstall ================="
#helm $HELM_ACTION $KUBE_VARIANT --keep-history 1>$KUBE_LOG_FILE 2>&1
helm $HELM_ACTION $KUBE_VARIANT  --namespace=$KUBE_NAMESPACE 1>$KUBE_LOG_FILE 2>&1

