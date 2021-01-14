#!/bin/bash 

QUERYID=$(aws logs start-query \
  --log-group-name CovidShield \
  --start-time 1610480596 \
  --end-time 1610484196 \
  --region "ca-central-1" \
  --query-string "fields @timestamp, @message
| sort @timestamp desc
| limit 20
| filter @message like /(?i)(msg=\"http response\")/
| filter @message like \"claim-key\"
| parse @message 'time=\"*\" level=* msg=\"*\" component=* headers=\"*\" path=* route=* statusClass=* statusCode=* uuid=*' as time, level, msg, component, headers, path, route, statusClass, statusCode, uuid
| parse statusClass \"2*\" as okayCode
| parse statusClass \"4*\" as unauthCode
| stats count(okayCode) as okayCodes, count(unauthCode) as unauthCodes, (count(okayCode)/(count(okayCode) + count(unauthCode))*100) as percentSuccess" | jq -r ".queryId")




function query { 
  QUERYID=$(aws logs start-query \
    --log-group-name CovidShield \
    --start-time "$1" \
    --end-time "$2" \
    --region "ca-central-1" \
    --query-string "fields @timestamp, @message
  | sort @timestamp desc
  | limit 20
  | filter @message like /(?i)(msg=\"http response\")/
  | filter @message like \"claim-key\"
  | parse @message 'time=\"*\" level=* msg=\"*\" component=* headers=\"*\" path=* route=* statusClass=* statusCode=* uuid=*' as time, level, msg, component, headers, path, route, statusClass, statusCode, uuid
  | parse statusClass \"2*\" as okayCode
  | parse statusClass \"4*\" as unauthCode
  | stats count(okayCode) as okayCodes, count(unauthCode) as unauthCodes, (count(okayCode)/(count(okayCode) + count(unauthCode))*100) as percentSuccess" | jq -r ".queryId")

  echo "$QUERYID" 

  while true; do
    RES=$(aws logs get-query-results \
      --region "ca-central-1" \
      --query-id "$QUERYID")

    echo "$RES"


    STATUS=$(jq -r ".status" <<< "$RES")

    echo Status of query "$STATUS"

    if [[ "$STATUS" == "Complete" ]]; then
      echo "20-12-$3, $(jq -r '.results[] | map(.value) | join(", ")' <<< "$RES")" >> results.csv
      return 0
    fi

    sleep 10
    echo Checking again
  done
} 


echo "Date, Okay Codes, Unauthorized Codes, Success Percentage"  >> results.csv
for i in {1..1};
do
  START=$(date  -v 2020y -v 12m -v "${i}"d -v 00H -v 00M -v 00S +%s)
  END=$(date  -v 2020y -v 12m -v "${i}"d -v 23H -v 59M -v 59S +%s)
  query "$START" "$END" "$i"
done
