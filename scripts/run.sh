#!/bin/bash
set -e
minikube start
eval $(minikube docker-env)

docker build -t jan-api-gateway:latest ./apps/jan-api-gateway
docker build -t jan-inference-model:latest ./apps/jan-inference-model

helm dependency update ./charts/umbrella-chart
helm install jan-server ./charts/umbrella-chart

kubectl port-forward svc/jan-server-jan-api-gateway 8080:8080
# helm uninstall jan-server
# check http://localhost:8080/api/swagger/index.html#/