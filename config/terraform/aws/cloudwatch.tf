resource "aws_cloudwatch_log_group" "covidshield" {
  name = var.cloudwatch_log_group_name

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_cloudwatch_metric_alarm" "retrieval_cpu_utilization_high" {
  alarm_name          = "retrieval-cpu-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs cpu utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_retrieval.name
  }
}

resource "aws_cloudwatch_metric_alarm" "submission_cpu_utilization_high" {
  alarm_name          = "submission-cpu-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs cpu utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_submission.name
  }
}

resource "aws_cloudwatch_metric_alarm" "retrieval_memory_utilization_high" {
  alarm_name          = "retrieval-memory-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs memory utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_retrieval.name
  }
}

resource "aws_cloudwatch_metric_alarm" "submission_memory_utilization_high" {
  alarm_name          = "submission-memory-utilization-high"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "MemoryUtilization"
  namespace           = "AWS/ECS"
  period              = "120"
  statistic           = "Average"
  threshold           = "50"
  alarm_description   = "This metric monitors ecs memory utilization"

  alarm_actions = [aws_sns_topic.alert_warning.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.covidshield.name
    ServiceName = aws_ecs_service.covidshield_key_submission.name
  }
}