export HELM_VALUE_FILE=$(pwd)/value.yaml
echo "Helm value file:" $HELM_VALUE_FILE

export LOG_DIR=$(pwd)/log
echo "Log file:" $LOG_DIR

export KUBE_VARIANT=dev
echo "Kube variant:" $KUBE_VARIANT

export KUBE_NAMESPACE=go-app
echo "Kube namespace:" $KUBE_NAMESPACE

export LOG_FILE=$(echo $LOG_DIR)/$(echo $KUBE_VARIANT)-helm-upgrade.log
echo "Log file:" $LOG_FILE

export HELM_DIR=../helm/app
echo "HELM directory:" $HELM_DIR

export HELM_VALUES=$(pwd)/values.yaml
echo "HELM values:" $HELM_VALUES

(cd .. && ./_script/helm-install.sh upgrade $KUBE_VARIANT $KUBE_NAMESPACE $LOG_DIR $HELM_DIR $HELM_VALUES 1>$LOG_FILE 2>&1)

