#!/bin/bash

mydir="$(dirname "$(which "$0")")"

# shellcheck source=./get-sg-ids.sh
source "$mydir"/get-sg-ids.sh

aws ec2 revoke-security-group-ingress \
  --group-id "$DB"  \
  --source-group "$VPN" \
  --region ca-central-1 \
  --port 3306 \
  --protocol tcp 1> /dev/null