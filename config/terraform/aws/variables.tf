###
# Global
###
variable "region" {
  type = string
}

variable "billing_tag_key" {
  type = string
}

variable "billing_tag_value" {
  type = string
}

###
# AWS Cloud Watch - cloudwatch.tf
###
variable "cloudwatch_log_group_name" {
  type = string
}

###
# AWS ECS - ecs.tf
###
variable "github_sha" {
  type    = string
  default = ""
}

variable "ecs_name" {
  type = string
}

variable "scale_in_cooldown" {
  type    = number
  default = null
}
variable "scale_out_cooldown" {
  type    = number
  default = null
}
variable "cpu_scale_metric" {
  type = number
  default = 60
}
variable "memory_scale_metric" {
  type = number
  default = 60
}
variable "min_capacity" {
  type    = number
  default = null
}
variable "max_capacity" {
  type    = number
  default = null
}

# Task Key Retrieval
variable "ecs_key_retrieval_name" {
  type = string
}

variable "ecs_task_key_retrieval_env_hmac_key" {
  type = string
}

variable "ecs_task_key_retrieval_env_ecdsa_key" {
  type = string
}

variable "retrieval_autoscale_enabled" {
  type = bool
}

# Task Key Submission
variable "ecs_key_submission_name" {
  type = string
}

variable "ecs_task_key_submission_env_key_claim_token" {
  type = string
}

# Metric provider
variable "metric_provider" {
  type = string
}

# Tracing provider
variable "tracer_provider" {
  type = string
}
variable "submission_autoscale_enabled" {
  type = bool
}

###
# AWS VPC - networking.tf
###
variable "vpc_cidr_block" {
  type = string
}

variable "vpc_name" {
  type = string
}

###
# AWS RDS - rds.tf
###
# RDS Subnet Group
variable "rds_db_subnet_group_name" {
  type = string
}

# RDS DB - Key Retrieval/Submission
variable "rds_server_db_name" {
  type = string
}

variable "rds_server_db_user" {
  type = string
}

variable "rds_server_db_password" {
  type = string
}

variable "rds_server_allocated_storage" {
  type = string
}

variable "rds_server_instance_class" {
  type = string
}

###
# AWS Route53 - route53.tf
###
variable "route53_zone_name" {
  type = string
}
