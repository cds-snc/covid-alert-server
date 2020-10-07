#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then
  echo "please set AWS_PROFILE"
  exit 0
fi

echo "Verifying not in prod..."
if [[ "$AWS_PROFILE" == "covid-prod" ]]; then 
  echo "WARNING THIS IS BEING RUN IN PROD, EXITING NOW!"
  exit 1
fi 

echo "using loginpath $AWS_PROFILE"

echo "CLEARING DIAGNOSIS KEYS"
mysql --login-path="$AWS_PROFILE"  << EOF
TRUNCATE TABLE server.diagnosis_keys
EOF

