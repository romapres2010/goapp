echo "Current directory:"
pwd

export LOG_FILE=$(pwd)/app-api-push.log

echo "Go to working directory:"
pushd ./../../

export APP_VERSION=$(cat ./deploy/version)
export APP_API_APP_NAME=$(cat ./deploy/app_api_app_name)
export APP_REPOSITORY=$(cat ./deploy/default_repository)

docker push $APP_REPOSITORY/$APP_API_APP_NAME:$APP_VERSION 1>$LOG_FILE 2>&1

echo "Go to current directory:"
popd