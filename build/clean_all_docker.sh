#!/usr/bin/env bash

docker ps -a  | awk '{print $1}' | xargs docker stop
docker ps -a  | awk '{print $1}' | xargs docker rm
