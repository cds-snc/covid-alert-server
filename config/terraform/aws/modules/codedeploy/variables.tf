variable "codedeploy_service_role_arn" {
  type        = string
  description = "The codedeploy service role arn"
}

variable "action_on_timeout" {
  type        = string
  description = "Action to take on deployment timeout"
}

variable "termination_wait_time_in_minutes" {
  type        = number
  description = "minutes to wait to terminate old deploy"
}

variable "cluster_name" {
  type        = string
  description = "The ECS cluster name"
}

variable "ecs_service_name" {
  type        = string
  description = "The ECS Service name."
}

variable "lb_listener_arns" {
  type        = list
  description = "List of Amazon Resource Names (ARNs) of the load balancer listeners."
}

variable "test_lb_listener_arns" {
  type        = list
  description = "List of Amazon Resource Names (ARNs) of the test load balancer listeners."
}

variable "aws_lb_target_group_blue_name" {
  type        = string
  description = "Name of the blue target group."
}

variable "aws_lb_target_group_green_name" {
  type        = string
  description = "Name of the green target group."
}

