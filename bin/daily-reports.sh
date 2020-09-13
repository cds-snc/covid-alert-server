#!/bin/bash

if [ -z "$AWS_PROFILE" ]; then 
  echo "please set AWS_PROFILE"
  exit 0
fi

echo "using loginpath $AWS_PROFILE"

echo "QUERY 1"
mysql --login-path="$AWS_PROFILE"  << EOF
SELECT DATE(ek.created), ek.originator, COUNT(*) 
FROM server.encryption_keys ek
WHERE ek.one_time_code IS NULL 
GROUP BY ek.originator, DATE(ek.created)
EOF

echo "QUERY 2"
mysql --login-path="$AWS_PROFILE" << EOF
SELECT server.diagnosis_keys.originator, count(*) FROM server.diagnosis_keys 
GROUP BY server.diagnosis_keys.originator
EOF