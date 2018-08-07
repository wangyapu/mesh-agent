#!/bin/bash

#ETCD_HOST=$(ip addr show docker0 | grep 'inet\b' | awk '{print $2}' | cut -d '/' -f 1)
ETCD_HOST=etcd
ETCD_PORT=2379
ETCD_URL=http://$ETCD_HOST:$ETCD_PORT

echo ETCD_URL = $ETCD_URL

if [[ "$1" == "consumer" ]]; then
  echo "Starting consumer agent..."
  /root/dists/consumer -mode=consumer -local-port=20000 -etcd-host=$ETCD_HOST -etcd-port=2379 -profile-dir=/root/logs/

elif [[ "$1" == "provider-small" ]]; then
  echo "Starting small provider agent..."
  /root/dists/provider -mode=provider -provider-port=30000 -provider-weight=300 -dubbo-port=20880 -etcd-host=$ETCD_HOST -etcd-port=2379 -profile-dir=/root/logs/

elif [[ "$1" == "provider-medium" ]]; then
  echo "Starting medium provider agent..."
  /root/dists/provider -mode=provider -provider-port=30000 -provider-weight=600 -dubbo-port=20880 -etcd-host=$ETCD_HOST -etcd-port=2379 -profile-dir=/root/logs/

elif [[ "$1" == "provider-large" ]]; then
  echo "Starting large provider agent..."
  /root/dists/provider -mode=provider -provider-port=30000 -provider-weight=600 -dubbo-port=20880 -etcd-host=$ETCD_HOST -etcd-port=2379 -profile-dir=/root/logs/

else
  echo "Unrecognized arguments, exit."
  exit 1
fi
