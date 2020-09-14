#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then 
  echo "please set AWS_PROFILE"
  exit 0
fi
MYDIR="$(dirname "$(which "$0")")"

# shellcheck source=./get-sg-ids.sh
source "$MYDIR"/get-sg-ids.sh

# Add Ingress
aws ec2 authorize-security-group-ingress \
  --protocol tcp \
  --group-id "$DB"  \
  --source-group "$VPN" \
  --region ca-central-1 \
  --port 3306 1> /dev/null
