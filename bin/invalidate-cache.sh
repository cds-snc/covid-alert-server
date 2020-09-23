#!/bin/bash 

ACCOUNT=$(aws sts get-caller-identity | jq '.Account' -r)

echo "Verifying not in prod..."
if [[ "$ACCOUNT" == 8* ]]; then
  echo "WARNING THIS IS BEING RUN IN PROD, EXITING NOW!"
  exit 1
fi


echo "Getting list of cloudfront distributions..."
DISTRIBUTIONS="$(aws cloudfront list-distributions)"

echo "Verifying number of cloudfront distributions..."
if [[ $(jq -r ".DistributionList.Items | length" <<< $DISTRIBUTIONS) -gt 1 ]]; then
  # This script was written when we only had 1 distribution
  echo "TOO MANY DISTRIBUTIONS PLEASE FIX SCRIPT"
  exit 1
fi

echo "Getting distribution ID"
DIST_ID="$(jq -r '.DistributionList.Items[0].Id' <<< $DISTRIBUTIONS)"

echo "$DIST_ID"
echo "Creating Invalidation ..."
aws cloudfront create-invalidation --distribution-id "$DIST_ID" --paths "/*" 