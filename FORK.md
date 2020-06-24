## Purpose

This purpose of this document is to track changes made compared to the original [CovidShield server](https://github.com/CovidShield/server) code.

### Setting the upstream

If you want to pull upstream merges to keep this repository in sync you can do that through the following `git` commands:

```
git remote add upstream git@github.com:CovidShield/server.git
git pull upstream master
```

or


```
git remote add upstream https://github.com/CovidShield/server.git
git pull upstream master
```

### Changes

- Added CI for Ruby unit tests (https://github.com/cds-snc/covid-shield-server/pull/3)
- Added DevContainer for VSCode (https://github.com/cds-snc/covid-shield-server/pull/7)
- Make CI files more generic (https://github.com/cds-snc/covid-shield-server/pull/9):
  - Moved Docker registry url and repository name to secrets
  - Moved retrieval/submission URLs to secrets
  - Removed hardcoded repository_owner
- Make Terraform files more generic (https://github.com/cds-snc/covid-shield-server/pull/12):
- Add private ECR Container repositories (https://github.com/cds-snc/covid-shield-server/pull/16)
- Replace MySQL with Aurora (https://github.com/cds-snc/covid-shield-server/pull/17)
- Added SNS topics to facilitate monitoring and reporting to outside tools (https://github.com/cds-snc/covid-shield-server/pull/42)
- Added basic cloudwatch alerts - more alerts to come later (https://github.com/cds-snc/covid-shield-server/pull/45)
- Added log metrics for key counts in the database (https://github.com/cds-snc/covid-shield-server/pull/51)
- Added `hashID` logic based on a provincial healthcare provider feature request (https://github.com/cds-snc/covid-shield-server/pull/50)
- Changing the code expiry window from 10 minutes to 24 hours (https://github.com/cds-snc/covid-shield-server/pull/56)
- Removed portal security group egress (https://github.com/cds-snc/covid-shield-server/pull/84)
