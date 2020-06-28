###
# Global
###

region = "ca-central-1"
# Enable the new ARN format to propagate tags to containers (see config/terraform/aws/README.md)
billing_tag_key   = "CostCentre"
billing_tag_value = "CovidShield"

###
# AWS Cloud Watch - cloudwatch.tf
###

cloudwatch_log_group_name = "CovidShield"

###
# AWS ECS - ecs.tf
###

ecs_name        = "CovidShield"
metric_provider = "stdout"
tracer_provider = "stdout"

# Key Retrieval
ecs_key_retrieval_name = "KeyRetrieval"
# Value should come from a TF_VAR environment variable (e.g. set in a Github Secret)
# ecs_task_key_retrieval_env_hmac_key = ""
# Value should come from a TF_VAR environment variable (e.g. set in a Github Secret)
# ecs_task_key_retrieval_env_ecdsa_key = ""

# Key Submission
ecs_key_submission_name = "KeySubmission"
# Value should come from a TF_VAR environment variable (e.g. set in a Github Secret)
# Must be a string of the form <secret1>=<MMC_code>:<secret2>=<MMC_code> - https://www.mcc-mnc.com
# ecs_task_key_submission_env_key_claim_token = ""

#Autoscaling ECS
submission_autoscale_enabled = true
retrieval_autoscale_enabled  = true
scale_down_adjustment        = -1
scale_up_adjustment          = 1
scale_down_cooldown          = 60
scale_up_cooldown            = 60
max_capacity                 = 10
min_capacity                 = 2
###
# AWS VPC - networking.tf
###

vpc_cidr_block = "10.0.0.0/16"
vpc_name       = "CovidShield"

###
# AWS RDS - rds.tf
###

rds_db_subnet_group_name = "server"

# Key Retrieval/Submission
rds_server_db_name = "server"
rds_server_db_user = "root"
# Value should come from a TF_VAR environment variable (e.g. set in a Github Secret)
# rds_server_db_password       = ""
rds_server_allocated_storage = "5"
rds_server_instance_class    = "db.r4.large"

###
# AWS Route 53 - route53.tf
###
# Value should come from a TF_VAR environment variable (e.g. set in a Github Secret)
# route53_zone_name = ""
