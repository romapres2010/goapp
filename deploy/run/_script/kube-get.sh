echo ""
echo "================================================="
echo "================= Get resources ================="

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

if [ -z "$2" ]
then
    echo "No namespace supplied"
    exit 1
fi

export KUBE_NAMESPACE=$2
echo "Kube namespace:" $KUBE_NAMESPACE

export KUBE_VARIANT=$1
echo "Kube variant:" $KUBE_VARIANT

echo ""
echo "================= kubectl get pods ================="
kubectl get pods -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get StatefulSet ================="
kubectl get StatefulSet -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get deployment ================="
kubectl get deployment -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get service ================="
kubectl get service -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get configmap ================="
kubectl get configmap -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get secret ================="
kubectl get secret -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get pvc ================="
kubectl get pvc -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get pv ================="
kubectl get pv -n $KUBE_NAMESPACE -o wide

echo ""
echo "================= kubectl get hpa ================="
kubectl get hpa -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl get ingress ================="
kubectl get ingress -n $KUBE_NAMESPACE -o wide -l variant=$KUBE_VARIANT
                    