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
