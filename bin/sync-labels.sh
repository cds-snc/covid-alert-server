#!/bin/bash
# github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server 
# github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-production-terraform
# github-label-sync -a "$GH_TOKEN"  -l labels.yml -d cds-snc/covid-alert-server-staging-terraform
# github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-demo-terraform
# github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-metrics-extractor
# github-label-sync -a "$GH_TOKEN" -l labels.yml -d cds-snc/covid-alert-server-metrics-extractor-creds

github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server 
github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-production-terraform
github-label-sync -a "$GH_TOKEN"  -l labels.yml cds-snc/covid-alert-server-staging-terraform
github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-demo-terraform
github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-metrics-extractor
github-label-sync -a "$GH_TOKEN" -l labels.yml cds-snc/covid-alert-server-metrics-extractor-creds