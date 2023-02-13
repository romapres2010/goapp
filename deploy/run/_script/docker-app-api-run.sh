then
    echo "No arguments supplied"
    exit 1
fi

if [ -z "$1" ]
then
    echo "No env file supplied"
    exit 1
else 
    echo "Env file:" $1
fi

echo "Current directory:"
pwd

echo "Go to working directory:"
pushd ./../../

export APP_VERSION=$(cat ./deploy/version)
export APP_API_APP_NAME=$(cat ./deploy/app_api_app_name)
export APP_REPOSITORY=$(cat ./deploy/default_repository)

echo Stop container docker-$APP_API_APP_NAME ...
docker stop docker-$APP_API_APP_NAME
echo Drop container $APP_API_APP_NAME ...
docker rm docker-$APP_API_APP_NAME

echo "Create folder for bind_mounts"
mkdir -p ./../app_volumes/cfg
mkdir -p ./../app_volumes/log

docker run -d \
  --name docker-$APP_API_APP_NAME \
  -p 8001:8080 \
  -p 3000:8080 \
  --env-file $1 \
  --mount type=bind,source="$(pwd)"/../app_volumes/cfg,target=/app/cfg,readonly \
  --mount type=bind,source="$(pwd)"/../app_volumes/log,target=/app/log \
  $APP_REPOSITORY/$APP_API_APP_NAME:$APP_VERSION

echo "Go to current directory:"
popd
