echo "Current directory:"
pwd

export LOG_FILE=$(pwd)/app-ui-build.log

echo "Go to working directory:"
pushd ./../../

export APP_VERSION=$(cat ./deploy/version)
export APP_UI_APP_NAME=$(cat ./deploy/app_ui_app_name)
export APP_REPOSITORY=$(cat ./deploy/default_repository)
export APP_BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
export APP_COMMIT=$(git rev-parse --short HEAD)

docker build --build-arg APP_VERSION=$APP_VERSION --build-arg APP_BUILD_TIME=$APP_BUILD_TIME --build-arg APP_COMMIT=$APP_COMMIT -t $APP_REPOSITORY/$APP_UI_APP_NAME:$APP_VERSION -f ./deploy/docker/app-ui.Dockerfile . 1>$LOG_FILE 2>&1

echo "Go to current directory:"
popd