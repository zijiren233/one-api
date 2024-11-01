#!/bin/bash
set -ex

kubectl create ns ai-system || true

kubectl create -f manifests/ai-proxy-config.yaml -n ai-system || true

kubectl apply -f manifests/deploy.yaml -n ai-system

if [[ -n "$cloudDomain" ]]; then
  kubectl create -f manifests/ingress.yaml -n ai-system || true
fi
