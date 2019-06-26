#!/usr/bin/env bash
kubectl -n cni-test delete deployment.extensions/prometheus service/prometheus pod/cni-e2e
