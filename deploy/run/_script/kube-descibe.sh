#https://kubernetes.io/ru/docs/reference/kubectl/cheatsheet/

echo ""
echo "======================================================"
echo "================= Describe resources ================="


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
echo "================= kubectl describe pods ================="
kubectl describe pods -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe deployment ================="
kubectl describe deployment -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe StatefulSet ================="
kubectl describe StatefulSet -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe service ================="
kubectl describe service -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe configmap ================="
kubectl describe configmap -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe secret ================="
kubectl describe secret -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe pvc ================="
kubectl describe pvc -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe pv ================="
kubectl describe pv -n $KUBE_NAMESPACE

echo ""
echo "================= kubectl describe hpa ================="
kubectl describe hpa -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT

echo ""
echo "================= kubectl describe ingress ================="
kubectl describe ingress -n $KUBE_NAMESPACE -l variant=$KUBE_VARIANT
                     