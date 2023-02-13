echo "Current directory:"
pwd

echo "Go to working directory:"
pushd ./../../

export APP_API_APP_NAME=$(cat ./deploy/app_api_app_name)

echo Stop container $APP_API_APP_NAME ...
docker stop docker-$APP_API_APP_NAME
echo Delete container $APP_API_APP_NAME ...
docker rm docker-$APP_API_APP_NAME

echo "Go to current directory:"
popd
