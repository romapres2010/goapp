if [ $# -eq 0 ]
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
export APP_LUQUIBASE_APP_NAME=$(cat ./deploy/app_liquibase_app_name)
export APP_REPOSITORY=$(cat ./deploy/default_repository)
export APP_BUILD_TIME=$(date '+%Y-%m-%d_%H:%M:%S')
export APP_COMMIT=$(git rev-parse --short HEAD)

echo "Create folder for bind_mounts"
mkdir -p ./../app_volumes/cfg
mkdir -p ./../app_volumes/log
mkdir -p ./../app_volumes/db

echo "Run compose"
docker compose --env-file $1 -f ./deploy/compose/compose-app.yaml up --build --detach $2 $3 $4 $5 $6 $7 $8

echo "Go to current directory:"
popd
