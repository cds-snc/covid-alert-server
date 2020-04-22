###
# Global
###

region            = "ca-central-1"
billing_tag_key   = "CostCentre"
billing_tag_value = "CovidShield"

###
# AWS Cloud Watch - cloudwatch.tf
###

cloudwatch_log_group_name = "CovidShield"

###
# AWS ECS - ecs.tf
###

ecs_name = "CovidShield"

# Key Retrieval
ecs_key_retrieval_name = "KeyRetrieval"
# Comes from a Github Secret
# ecs_task_key_retrieval_env_hmac_key = ""

# Key Submission
ecs_key_submission_name = "KeySubmission"
# Must be a string of the form <secret1>=<region_id1>:<secret2=region_id2> etc.
# Comes from a Github Secret
# ecs_task_key_submission_env_key_claim_token = ""

###
# AWS VPC - networking.tf
###

vpc_cidr_block = "10.0.0.0/16"
vpc_name       = "CovidShield"

###
# AWS RDS - rds.tf
###

rds_db_subnet_group_name = "backend"

# Key Retrieval/Submission
rds_backend_db_name = "backend"
rds_backend_db_user = "root"
# Comes from a Github Secret
# rds_backend_db_password       = ""
rds_backend_allocated_storage = "5"
rds_backend_instance_class    = "db.t3.small"

###
# AWS Route 53 - route53.tf
###

route53_zone_name = "covidshield.app"
