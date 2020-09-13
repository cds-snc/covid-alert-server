#!/bin/bash

function getSG { 
  aws ec2 describe-security-groups --region ca-central-1 | \
  jq ".SecurityGroups[] | select(.GroupName == \"$1\") | .GroupId"  -r
}

VPN="$(getSG "VPN_SG")"
export VPN

DB="$(getSG "covidshield-database")"
export DB

