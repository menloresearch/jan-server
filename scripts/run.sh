#!/bin/bash
set -e
minikube start
eval $(minikube docker-env)

docker build -t menloltd/jan-server:latest ./apps/jan-api-gateway

helm dependency update ./charts/jan-server
helm install jan-server ./charts/jan-server --set gateway.image.tag=latest

kubectl port-forward svc/jan-server-jan-api-gateway 8080:8080
# helm uninstall jan-server
# check http://localhost:8080/api/swagger/index.html#/