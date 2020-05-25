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

data "github_branch" "backend" {
  repository = "backend"
  branch     = "master"
}

###
# ECS - Key Retrieval
###

# Task Definition

data "template_file" "covidshield_key_retrieval_task" {
  template = file("task-definitions/covidshield_key_retrieval.json")

  vars = {
    image                 = "covidshield/key-retrieval:${coalesce(var.github_sha, data.github_branch.backend.sha)}"
    awslogs-group         = aws_cloudwatch_log_group.covidshield.name
    awslogs-region        = var.region
    awslogs-stream-prefix = "ecs-${var.ecs_key_retrieval_name}"
    retrieve_hmac_key     = aws_secretsmanager_secret_version.key_retrieval_env_hmac_key.arn
    ecdsa_key             = aws_secretsmanager_secret_version.key_retrieval_env_ecdsa_key.arn
    database_url          = aws_secretsmanager_secret_version.backend_database_url.arn
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

  name            = var.ecs_key_retrieval_name
  cluster         = aws_ecs_cluster.covidshield.id
  task_definition = aws_ecs_task_definition.covidshield_key_retrieval.arn
  desired_count   = 2
  launch_type     = "FARGATE"

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
}


###
# ECS - Key Submission
###

# Task Definition

data "template_file" "covidshield_key_submission_task" {
  template = file("task-definitions/covidshield_key_submission.json")

  vars = {
    image                 = "covidshield/key-submission:${coalesce(var.github_sha, data.github_branch.backend.sha)}"
    awslogs-group         = aws_cloudwatch_log_group.covidshield.name
    awslogs-region        = var.region
    awslogs-stream-prefix = "ecs-${var.ecs_key_submission_name}"
    key_claim_token       = aws_secretsmanager_secret_version.key_submission_env_key_claim_token.arn
    database_url          = aws_secretsmanager_secret_version.backend_database_url.arn
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

  name            = var.ecs_key_submission_name
  cluster         = aws_ecs_cluster.covidshield.id
  task_definition = aws_ecs_task_definition.covidshield_key_submission.arn
  desired_count   = 2
  launch_type     = "FARGATE"

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
}
