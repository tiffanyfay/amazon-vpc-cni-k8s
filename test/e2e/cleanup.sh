#!/usr/bin/env bash
kubectl -n cni-test delete deployment.extensions/prometheus-deployment service/prometheus deployment/cni-e2e
