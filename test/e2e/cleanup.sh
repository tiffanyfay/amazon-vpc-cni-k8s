#!/usr/bin/env bash
kubectl -n cni-test delete deployment.extensions/prometheus service/prometheus deployment/cni-e2e
