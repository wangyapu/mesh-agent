#!/usr/bin/env bash

docker run -d --name=etcd --memory=1g --network=mesh registry.cn-hangzhou.aliyuncs.com/aliware2018/alpine-etcd
docker run -d --name=provider-small --cpu-period=50000 --cpu-quota=30000  --memory=2g  --network=mesh mesh_agent:latest provider-small
docker run -d --name=provider-medium --cpu-period=50000 --cpu-quota=60000 --memory=4g --network=mesh mesh_agent:latest provider-medium
docker run -d --name=provider-large  --cpu-period=50000 --cpu-quota=90000 --memory=6g --network=mesh mesh_agent:latest provider-large
docker run -d -p 8087:8087 -p 20000:20000 --cpu-period 50000 --cpu-quota 180000 --memory=3g --network=mesh --name=consumer mesh_agent:latest consumer
