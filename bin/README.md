# TOIL


## docker-aws-login.sh

Logs into the ECR registry in ca-central-1 region

## get-sg-ids.sh

Used to get the security group names we need

## add-vpn-nsg.sh

Add the VPN_SG security group to the covidshield-database security group

## daily-reports.sh

Run the sql that we currently use to generate daily reports

**Please note: This requires a login-path to be setup and using the same name as the aws credential you are using**

## remove-vpn-nsg.sh

Removes the VPN_SG security group from the covidshield-database security group