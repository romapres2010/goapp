if [ $# -eq 0 ]
then
    echo "No arguments supplied"
    exit 1
fi

if [ -z "$1" ]
then
    echo "No kibe variant supplied"
    exit 1
fi

export KUBE_NAMESPACE=go-app
echo "Kube namespace:" $KUBE_NAMESPACE

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

export YAML_FILE=$(pwd)/log/build-$(echo $KUBE_VARIANT)-kustomize.yaml
echo "YAML file:" $YAML_FILE

export KUSTOMIZE_LOG_FILE=$(pwd)/log/build-$(echo $KUBE_VARIANT)-kustomize.log
echo "Kustomize Log file:" $KUSTOMIZE_LOG_FILE

export KUBE_LOG_FILE=$(pwd)/log/build-$(echo $KUBE_VARIANT)-kube.log
echo "Kube Log file:" $KUBE_LOG_FILE

export KUBE_DESCRIBE_FILE=$(pwd)/log/describe-$(echo $KUBE_VARIANT)-kube.log
echo "APP describe file:" $KUBE_DESCRIBE_FILE

echo "kubectl kustomize ./overlays/$(echo $KUBE_VARIANT)"
kubectl kustomize ./overlays/$KUBE_VARIANT 1>$YAML_FILE 2>$KUSTOMIZE_LOG_FILE

echo "kubectl apply"
kubectl apply -f $YAML_FILE 1>$KUBE_LOG_FILE 2>&1

./kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>$KUBE_DESCRIBE_FILE 2>&1

./kube-descibe.sh $KUBE_VARIANT $KUBE_NAMESPACE 1>>$KUBE_DESCRIBE_FILE  2>&1

./kube-get.sh $KUBE_VARIANT $KUBE_NAMESPACE

if [ -z "$2" ]
then
    echo "sleep: 120"
    sleep 120
else
    echo "sleep:" $2 
    sleep $2
fi

./kube-log.sh $KUBE_VARIANT $KUBE_NAMESPACE

