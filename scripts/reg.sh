#!/bin/bash

# create a directory to store user passwords
mkdir -p ./scripts/registry_auth

# use htpasswd to create an encrypted file
htpasswd -Bbn test 1234 > ./scripts/registry_auth/htpasswd

# check if there is a container named registry
if [ "$(docker ps -aq -f name=kcl-registry)" ]; then
    # stop and remove the container named registry
    docker stop kcl-registry
    docker rm kcl-registry
fi

docker network rm kcl || true
docker network create kcl --subnet=172.88.0.0/24

# start the Docker Registry with authentication
docker run -p 5002:5002 \
--restart=always \
--name kcl-registry \
--network kcl \
--ip 172.88.0.8 \
-v /var/lib/registry:/var/lib/registry \
-v $PWD/scripts/registry_auth/:/auth/ \
-e "REGISTRY_AUTH=htpasswd" \
-e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
-e "REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd" \
-e "REGISTRY_HTTP_ADDR=:5002" \
-d registry

# clean the registry
docker exec kcl-registry rm -rf /var/lib/registry/docker/registry/v2/repositories/
