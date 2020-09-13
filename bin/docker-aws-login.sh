#!/bin/bash

ACCOUNT=$(aws sts get-caller-identity | jq '.Account' -r)

aws ecr get-login-password --region ca-central-1 | docker login --username AWS --password-stdin "$ACCOUNT".dkr.ecr.ca-central-1.amazonaws.com
