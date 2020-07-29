resource "aws_codedeploy_app" "app" {
  compute_platform = "ECS"
  name             = "AppECS-${var.cluster_name}-${var.ecs_service_name}"
}

resource "aws_codedeploy_deployment_group" "app" {
  app_name               = aws_codedeploy_app.app.name
  deployment_config_name = "CodeDeployDefault.ECSAllAtOnce"
  deployment_group_name  = "DgpECS-${var.cluster_name}-${var.ecs_service_name}"
  service_role_arn       = var.codedeploy_service_role_arn

  auto_rollback_configuration {
    enabled = true
    events  = ["DEPLOYMENT_FAILURE"]
  }

  deployment_style {
    deployment_option = "WITH_TRAFFIC_CONTROL"
    deployment_type   = "BLUE_GREEN"
  }

  blue_green_deployment_config {
    deployment_ready_option {
      action_on_timeout = var.action_on_timeout
    }

    terminate_blue_instances_on_deployment_success {
      action                           = "TERMINATE"
      termination_wait_time_in_minutes = var.termination_wait_time_in_minutes
    }
  }



  ecs_service {
    cluster_name = var.cluster_name
    service_name = var.ecs_service_name
  }

  load_balancer_info {
    target_group_pair_info {
      prod_traffic_route {
        listener_arns = var.lb_listener_arns
      }

      target_group {
        name = var.aws_lb_target_group_blue_name
      }

      target_group {
        name = var.aws_lb_target_group_green_name
      }

      test_traffic_route {
        listener_arns = var.test_lb_listener_arns
      }
    }
  }
}
