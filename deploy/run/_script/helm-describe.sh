#https://kubernetes.io/ru/docs/reference/kubectl/cheatsheet/

if [ $# -eq 3 ]
then
    echo "Got 3 arguments: [KUBE_VARIANT:$1], [KUBE_NAMESPACE:$2], [LOG_DIR:$3]"
else 
    echo "No enough arguments supplied"
    exit 1
fi

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

export KUBE_NAMESPACE=$2
echo "Kube namespace:" $KUBE_NAMESPACE

export LOG_DIR=$3
echo "Log directory:" $LOG_DIR

export KUBE_DESCRIBE_FILE=$(echo $LOG_DIR)/helm-describe-$(echo $KUBE_VARIANT).log
echo "Describe file:" $KUBE_DESCRIBE_FILE

echo "================= RUN kube-get.sh ================="
./_script/kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>$KUBE_DESCRIBE_FILE 2>&1

echo "================= RUN kube-descibe.sh ================="
./_script/kube-descibe.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>>$KUBE_DESCRIBE_FILE  2>&1

echo "================= RUN kube-log.sh ================="
./_script/kube-log.sh $KUBE_VARIANT $KUBE_NAMESPACE $LOG_DIR
