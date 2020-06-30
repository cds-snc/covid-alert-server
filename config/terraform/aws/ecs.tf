###
# ECS Cluster
###

resource "aws_ecs_cluster" "covidshield" {
  name = var.ecs_name

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

data "github_branch" "server" {
  repository = "covid-shield-server"
  branch     = "master"
}

locals {
  retrieval_repo  = values(aws_ecr_repository.repository)[0].repository_url
  submission_repo = values(aws_ecr_repository.repository)[1].repository_url
}

###
# ECS - Key Retrieval
###

# Task Definition

data "template_file" "covidshield_key_retrieval_task" {
  template = file("task-definitions/covidshield_key_retrieval.json")

  vars = {
    image                 = "${local.retrieval_repo}:${coalesce(var.github_sha, data.github_branch.server.sha)}"
    awslogs-group         = aws_cloudwatch_log_group.covidshield.name
    awslogs-region        = var.region
    awslogs-stream-prefix = "ecs-${var.ecs_key_retrieval_name}"
    retrieve_hmac_key     = aws_secretsmanager_secret_version.key_retrieval_env_hmac_key.arn
    ecdsa_key             = aws_secretsmanager_secret_version.key_retrieval_env_ecdsa_key.arn
    database_url          = aws_secretsmanager_secret_version.server_database_url.arn
    metric_provider       = var.metric_provider
    tracer_provider       = var.tracer_provider
  }
}

resource "aws_ecs_task_definition" "covidshield_key_retrieval" {
  family       = var.ecs_key_retrieval_name
  cpu          = 2048
  memory       = "4096"
  network_mode = "awsvpc"

  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.covidshield_key_retrieval.arn
  task_role_arn            = aws_iam_role.covidshield_key_retrieval.arn
  container_definitions    = data.template_file.covidshield_key_retrieval_task.rendered

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

# Service

resource "aws_ecs_service" "covidshield_key_retrieval" {
  depends_on = [
    aws_lb_listener.covidshield_key_retrieval,
  ]

  name             = var.ecs_key_retrieval_name
  cluster          = aws_ecs_cluster.covidshield.id
  task_definition  = aws_ecs_task_definition.covidshield_key_retrieval.arn
  launch_type      = "FARGATE"
  platform_version = "1.4.0"
  # Enable the new ARN format to propagate tags to containers (see config/terraform/aws/README.md)
  propagate_tags = "SERVICE"

  desired_count                      = 2
  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 60

  network_configuration {
    assign_public_ip = false
    subnets          = aws_subnet.covidshield_private.*.id
    security_groups = [
      aws_security_group.covidshield_egress_anywhere.id,
      aws_security_group.covidshield_key_retrieval.id,
    ]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.covidshield_key_retrieval.arn
    container_name   = "key-retrieval"
    container_port   = 8001
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_appautoscaling_target" "retrieval" {
  count              = var.retrieval_autoscale_enabled ? 1 : 0
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_retrieval.cluster}/${aws_ecs_service.covidshield_key_retrieval.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.min_capacity
  max_capacity       = var.max_capacity
}
resource "aws_appautoscaling_policy" "retrieval_cpu" {
  count              = var.retrieval_autoscale_enabled ? 1 : 0
  name               = "retrieval_cpu"
  policy_type        = "TargetTrackingScaling"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_retrieval.cluster}/${aws_ecs_service.covidshield_key_retrieval.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  target_tracking_scaling_policy_configuration {
    scale_in_cooldown  = var.scale_in_cooldown
    scale_out_cooldown = var.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value = var.cpu_scale_metric
  }
}

resource "aws_appautoscaling_policy" "retrieval_memory" {
  count              = var.retrieval_autoscale_enabled ? 1 : 0
  name               = "retrieval_memory"
  policy_type        = "TargetTrackingScaling"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_retrieval.cluster}/${aws_ecs_service.covidshield_key_retrieval.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  target_tracking_scaling_policy_configuration {
    scale_in_cooldown  = var.scale_in_cooldown
    scale_out_cooldown = var.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
    target_value = var.memory_scale_metric
  }
}

###
# ECS - Key Submission
###

# Task Definition

data "template_file" "covidshield_key_submission_task" {
  template = file("task-definitions/covidshield_key_submission.json")

  vars = {
    image                 = "${local.submission_repo}:${coalesce(var.github_sha, data.github_branch.server.sha)}"
    awslogs-group         = aws_cloudwatch_log_group.covidshield.name
    awslogs-region        = var.region
    awslogs-stream-prefix = "ecs-${var.ecs_key_submission_name}"
    key_claim_token       = aws_secretsmanager_secret_version.key_submission_env_key_claim_token.arn
    database_url          = aws_secretsmanager_secret_version.server_database_url.arn
    metric_provider       = var.metric_provider
    tracer_provider       = var.tracer_provider
  }
}

resource "aws_ecs_task_definition" "covidshield_key_submission" {
  family       = var.ecs_key_submission_name
  cpu          = 2048
  memory       = "4096"
  network_mode = "awsvpc"

  requires_compatibilities = ["FARGATE"]
  execution_role_arn       = aws_iam_role.covidshield_key_submission.arn
  task_role_arn            = aws_iam_role.covidshield_key_submission.arn
  container_definitions    = data.template_file.covidshield_key_submission_task.rendered

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

# Service

resource "aws_ecs_service" "covidshield_key_submission" {
  depends_on = [
    aws_lb_listener.covidshield_key_submission,
  ]

  name             = var.ecs_key_submission_name
  cluster          = aws_ecs_cluster.covidshield.id
  task_definition  = aws_ecs_task_definition.covidshield_key_submission.arn
  launch_type      = "FARGATE"
  platform_version = "1.4.0"
  # Enable the new ARN format to propagate tags to containers (see config/terraform/aws/README.md)
  propagate_tags = "SERVICE"

  desired_count                      = 2
  deployment_minimum_healthy_percent = 50
  deployment_maximum_percent         = 200
  health_check_grace_period_seconds  = 60

  network_configuration {
    assign_public_ip = false
    subnets          = aws_subnet.covidshield_private.*.id
    security_groups = [
      aws_security_group.covidshield_egress_anywhere.id,
      aws_security_group.covidshield_key_submission.id,
    ]
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.covidshield_key_submission.arn
    container_name   = "key-submission"
    container_port   = 8000
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}
resource "aws_appautoscaling_target" "submission" {
  count              = var.submission_autoscale_enabled ? 1 : 0
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_submission.cluster}/${aws_ecs_service.covidshield_key_submission.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  min_capacity       = var.min_capacity
  max_capacity       = var.max_capacity
}
resource "aws_appautoscaling_policy" "submission_cpu" {
  count              = var.submission_autoscale_enabled ? 1 : 0
  name               = "submission_cpu"
  policy_type        = "TargetTrackingScaling"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_submission.cluster}/${aws_ecs_service.covidshield_key_submission.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  target_tracking_scaling_policy_configuration {
    scale_in_cooldown  = var.scale_in_cooldown
    scale_out_cooldown = var.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value = var.cpu_scale_metric
  }
}

resource "aws_appautoscaling_policy" "submission_memory" {
  count              = var.submission_autoscale_enabled ? 1 : 0
  name               = "submission_memory"
  policy_type        = "TargetTrackingScaling"
  service_namespace  = "ecs"
  resource_id        = "service/${aws_ecs_service.covidshield_key_submission.cluster}/${aws_ecs_service.covidshield_key_submission.name}"
  scalable_dimension = "ecs:service:DesiredCount"

  target_tracking_scaling_policy_configuration {
    scale_in_cooldown  = var.scale_in_cooldown
    scale_out_cooldown = var.scale_out_cooldown
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageMemoryUtilization"
    }
    target_value = var.memory_scale_metric
  }
}
