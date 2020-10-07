#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then
  echo "please set AWS_PROFILE"
  exit 0
fi

echo "using loginpath $AWS_PROFILE"

DATE=$(date -v-1d +%F)

echo "Events"
mysql --login-path="$AWS_PROFILE" << EOF
SELECT * FROM server.events WHERE date = "$DATE"
EOF
