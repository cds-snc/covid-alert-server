check_endpoint () {
  local deployed_revision
  deployed_revision=$(curl -s "https://${1}/services/version.json" | jq -r .revision)
  if [ "$deployed_revision" == ${2} ]; then
    return 0
  fi
return 1
}
verify_deployments () {
  set +m
  local OPTIND c e COMMIT_SHA ENDPOINTS
  while getopts ":c:e:" opt; do
    case "$opt" in
      c) COMMIT_SHA="$OPTARG"
      ;;
      e) ENDPOINTS+=("$OPTARG")
      ;;
      \?) echo "Invalid option -$OPTARG" >&2
      ;;
      :) echo "Option -$OPTARG requires an argument."; return 1
      ;;
    esac
  done
  shift $((OPTIND -1))
  count=0
  for endpoint in "${ENDPOINTS[@]}"
  do
    endpoint_count=0
     while [[ "$count" -le 180 && ("$endpoint_count" -lt 5) ]]
      do
        echo "Watching for new deployments in $endpoint - loop ${count}"
        check_endpoint $endpoint $COMMIT_SHA
        if [[ $? == 0 ]]; then
          endpoint_count=$(($endpoint_count+1))
          if [[ "$endpoint_count" == 1 ]]; then
            echo "$endpoint was successfully updated - $COMMIT_SHA"
          fi
        fi
        count=$(( $count + 1 ))
        sleep 1
        if [ $count -eq 10 ]; then
        echo "Deployment failed"
          return 1
        fi
     done & 
  done
  while [[ -n $(jobs -r) ]]; do sleep 5; done
}
