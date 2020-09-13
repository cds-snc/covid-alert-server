#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then 
  echo "please set AWS_PROFILE"
  exit 0
fi

source get-sg-ids.sh

aws ec2 revoke-security-group-ingress \
  --group-id "$DB"  \
  --source-group "$VPN" \
  --region ca-central-1 \
  --port 3306 \
  --protocol tcp 1> /dev/null