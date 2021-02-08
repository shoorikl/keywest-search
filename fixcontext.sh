#!/bin/sh


if [ $CF_BRANCH == "master" ]
then
    export CONTEXT="www"
    export REPLICAS=1
else
    export CONTEXT="${CF_BRANCH}"
    export REPLICAS=1
fi

printenv

# echo "CONTEXT=${CONTEXT}"
# #export TAG="${CICD_EXECUTION_SEQUENCE}-${CICD_GIT_BRANCH}-${CICD_GIT_COMMIT}"

# sed -i 's^${CONTEXT}^'"$CONTEXT^g" deployment.yaml
# sed -i 's^${REPLICAS}^'"$REPLICAS^g" deployment.yaml
# #sed -i 's^${TAG}^'"$TAG^g" deployment.yaml
# sed -i 's^${CICD_EXECUTION_SEQUENCE}^'"$CICD_EXECUTION_SEQUENCE^g" ./src/*.go
# sed -i 's^${CICD_EXECUTION_ID}^'"$CICD_EXECUTION_ID^g" ./src/*.go
# sed -i 's^${CICD_PIPELINE_ID}^'"$CICD_PIPELINE_ID^g" ./src/*.go
# sed -i 's^${CICD_GIT_BRANCH}^'"$CICD_GIT_BRANCH^g" ./src/*.go
# sed -i 's^${CICD_GIT_COMMIT}^'"$CICD_GIT_COMMIT^g" ./src/*.go
# sed -i 's^${CICD_GIT_REPO_NAME}^'"$CICD_GIT_REPO_NAME^g" ./src/*.go

# cat deployment.yaml