#!/bin/bash

if [[ -z "${SMEE_URL}" ]]; then 
    echo "SMEE_URL must be specified"
    exit -1
fi

smee() {
    while :
    do
        nohup smee -u "${SMEE_URL}" --path /webhook & > /dev/null
        sleep 3600
        pkill node 
        echo Killed smee
    done
}


cleanup() {
    pkill node
}

smee &
trap 'cleanup' SIGINT &

./server
