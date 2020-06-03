# Deploying in Kubernetes

## Pre-requisites

### 1. Run Kubernetes Cluster

The only pre-requisite is to run Kubernetes.

There are many ways to run Kubernetes, whether it is on the cloud or locally. 

You can use any of the following to run Kubernetes in your preferred environment:

[Minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/)

[GKE](https://cloud.google.com/kubernetes-engine)

[EKS](https://aws.amazon.com/eks/)

[AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/)

### 2. Install Nginx Ingress Controller

Install Nginx Ingress Controller https://github.com/kubernetes/ingress-nginx

## Deploying COVID Shield to a Kubernetes Cluster

`kubectl apply -f ./deploy/kubernetes`

## Adding Kubernetes Secrets

`kubectl create secret generic covidshield-secret --from-env-file=./deploy/kubernetes/secrets.development.env -n covidshield`

## Interacting with COVID Shield

View logs for key-submission Pod:

`kubectl logs -f key-submission-xxxxxx`

View the IP of the COVID Shield Ingress:

```
kubectl get ingress
NAME          CLASS    HOSTS   ADDRESS        PORTS   AGE
covidshield   <none>   *       192.168.64.2   80      23m
```

Call a route in the COVID Shield application:

`curl $INGRESS_IP/claim-key`

## Running in Minikube 

Run this command in order to be able to pull images from your local Docker registry in Minikube: 

`eval $(minikube docker-env)`

Enable Ingress Controller: 

`minikube addons enable ingress`

View logs for key-submission Pod:

`kubectl logs -f key-submission-xxxxxx`

Call a route in the COVID Shield application:

`export MINIKUBE_IP=`minikube ip``
`curl $MINIKUBE_IP/claim-key`

## Deployment Notes

- Specifying a namespace in the `PersistentVolumeClaim` may not be necessary in environments outside of Minikube.
- Expect for the `key-retrieval` Deployment to fail to start for ~30 seconds while the `mysql` Deployment spins up.
- You must create the Kubernetes secret mentioned above for the pods to fully spin up.
- Do not commit the `secrets.development.env` file to version control. This file contains secrets and only exists in this repository for the reference implementation as an example.
