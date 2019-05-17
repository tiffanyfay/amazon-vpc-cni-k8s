#!/usr/bin/env bash
kubectl -n cni-test delete namespace/cni-test deployment.extensions/prometheus-deployment service/prometheus 
#serviceaccount/testpod deployment.extensions/testpod service/testpod-clusterip service/testpod-pod-ip