#!/usr/bin/env bash
kubectl -n cni-test delete deployment.extensions/prometheus-deployment service/prometheus 
#serviceaccount/testpod deployment.extensions/testpod service/testpod-clusterip service/testpod-pod-ip