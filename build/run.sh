#!/usr/bin/env bash

docker build -t mesh_agent .
./build/clean_all_docker.sh
./build/start_test_docker.sh
