#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then 
  echo "please set AWS_PROFILE"
  exit 0
fi

source get-sg-ids.sh
echo "$DB"
echo "$VPN"
# Add Ingress
aws ec2 authorize-security-group-ingress \
  --protocol tcp \
  --group-id "$DB"  \
  --source-group "$VPN_SG" \
  --region ca-central-1 \
  --port 3306 