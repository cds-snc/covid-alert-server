#!/bin/bash

if [[ $# -eq 0 ]]; then 
  echo "sync-labels dry-run"
  echo "  view actions that will be taken"
  echo "sync-labels apply"
  echo "  apply changes to labels"
  exit 1
fi 

if [[ $1 == "dry-run" ]]; then
  github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server
  github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-production-terraform
  github-label-sync -a "$GH_TOKEN"  -l labels.yml -d cds-snc/covid-alert-server-staging-terraform
  github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-demo-terraform
  github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-metrics-extractor
  github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-metrics-extractor-creds
  exit 0
fi

if [[ $1 == "apply" ]]; then
  github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server
  github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-production-terraform
  github-label-sync -a "$GH_TOKEN"  -l labels.yml cds-snc/covid-alert-server-staging-terraform
  github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-demo-terraform
  github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-metrics-extractor
  github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-metrics-extractor-creds
  exit 0
fi