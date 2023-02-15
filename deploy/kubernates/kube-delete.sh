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

export KUBE_NAMESPACE=go-app
echo "Kube namespace:" $KUBE_NAMESPACE

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

export KUBE_LOG_FILE=$(pwd)/log/delete-$(echo $KUBE_VARIANT)-kube.log
echo "Kube Log file:" $KUBE_LOG_FILE

export KUBE_DESCRIBE_FILE=$(pwd)/log/describe-$(echo $KUBE_VARIANT)-kube.log
echo "APP describe file:" $KUBE_DESCRIBE_FILE

./kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>$KUBE_DESCRIBE_FILE 2>&1

./kube-descibe.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>>$KUBE_DESCRIBE_FILE  2>&1

./kube-log.sh $KUBE_VARIANT $KUBE_NAMESPACE 

echo "" 1>$KUBE_LOG_FILE 2>&1
echo "Delete all resources" 1>>$KUBE_LOG_FILE 2>&1

echo "kubectl delete pods,services,deployments -n app -l app=app,variant=$KUBE_VARIANT"
kubectl delete pods,services,deployments -n $KUBE_NAMESPACE -l app=app,variant=$KUBE_VARIANT 1>>$KUBE_LOG_FILE 2>&1

echo ""
echo "kubectl delete configmap -n app -l app=app"
kubectl delete configmap -n $KUBE_NAMESPACE -l app=app 1>>$KUBE_LOG_FILE 2>&1

echo ""
echo "kubectl delete secret -n app -l app=app"
kubectl delete secret -n $KUBE_NAMESPACE -l app=app 1>>$KUBE_LOG_FILE 2>&1

echo ""
echo "kubectl delete NetworkPolicy -n app app-net"
kubectl delete NetworkPolicy -n $KUBE_NAMESPACE app-net 1>>$KUBE_LOG_FILE 2>&1

echo ""
echo "kubectl delete namespace app"
kubectl delete namespace $KUBE_NAMESPACE 1>>$KUBE_LOG_FILE 2>&1

echo "" 1>>$KUBE_LOG_FILE 2>&1
echo "Check resources" 1>>$KUBE_DESCRIBE_FILE 2>&1
./kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>>$KUBE_DESCRIBE_FILE 2>&1

echo ""
./kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 


