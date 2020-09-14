#!/bin/bash

mydir="$(dirname "$(which "$0")")"

# shellcheck source=./get-sg-ids.sh
source "$mydir"/get-sg-ids.sh

# Add Ingress
aws ec2 authorize-security-group-ingress \
  --protocol tcp \
  --group-id "$DB"  \
  --source-group "$VPN" \
  --region ca-central-1 \
  --port 3306 1> /dev/null
