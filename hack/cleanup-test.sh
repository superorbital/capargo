#!/bin/sh

kubectl delete application -n argocd guestbook || true
kubectl delete cluster -n vcluster kind || true
