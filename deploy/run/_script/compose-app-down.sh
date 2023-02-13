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
export APP_BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
export APP_COMMIT=$(git rev-parse --short HEAD)

echo "Down compose"
docker compose --env-file $1 -f ./deploy/compose/compose-app.yaml down

echo "Go to current directory:"
popd

