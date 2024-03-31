#!/bin/sh

set -e

kubectl create ns argocd

sleep 2

kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

kubectl rollout -n argocd status deployment argocd-server

clusterctl init --infrastructure vcluster

kubectl rollout -n capi-system status deployment argocd-server

kubectl create namespace vcluster
