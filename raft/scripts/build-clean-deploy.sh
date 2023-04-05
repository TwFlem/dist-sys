#!/bin/bash
podman build -t raft .  
podman push localhost/raft:latest docker://docker.io/twflem/raft:latest
kubectl delete -f k8s
kubectl apply -f k8s
