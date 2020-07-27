module "covid-shield-retrieval" {
  source                           = "./modules/codedeploy"
  codedeploy_service_role_arn      = aws_iam_role.codedeploy.arn
  action_on_timeout                = var.manual_deploy_enabled ? "STOP_DEPLOYMENT" : "CONTINUE_DEPLOYMENT"
  termination_wait_time_in_minutes = var.termination_wait_time_in_minutes
  cluster_name                     = aws_ecs_cluster.covidshield.name
  ecs_service_name                 = aws_ecs_service.covidshield_key_retrieval.name
  lb_listener_arns                 = [aws_lb_listener.covidshield_key_retrieval.arn]
  test_lb_listener_arns            = [aws_lb_listener.covidshield_key_retrieval_test.arn]
  aws_lb_target_group_blue_name    = aws_lb_target_group.covidshield_key_retrieval.name
  aws_lb_target_group_green_name   = aws_lb_target_group.covidshield_key_retrieval_2.name
}

module "covid-shield-submission" {
  source                           = "./modules/codedeploy"
  codedeploy_service_role_arn      = aws_iam_role.codedeploy.arn
  action_on_timeout                = var.manual_deploy_enabled ? "STOP_DEPLOYMENT" : "CONTINUE_DEPLOYMENT"
  termination_wait_time_in_minutes = var.termination_wait_time_in_minutes
  cluster_name                     = aws_ecs_cluster.covidshield.name
  ecs_service_name                 = aws_ecs_service.covidshield_key_submission.name
  lb_listener_arns                 = [aws_lb_listener.covidshield_key_submission.arn]
  test_lb_listener_arns            = [aws_lb_listener.covidshield_key_submission_test.arn]
  aws_lb_target_group_blue_name    = aws_lb_target_group.covidshield_key_submission.name
  aws_lb_target_group_green_name   = aws_lb_target_group.covidshield_key_submission_2.name
}

###
# AWS Lambda - Validate Deploy
###
data "archive_file" "lambda_validate_deploy" {
  type        = "zip"
  source_file = "lambda/lambda_validate_deploy.rb"
  output_path = "/tmp/lambda_validate_deploy.rb.zip"
}

resource "aws_lambda_function" "retrieval_lambda_validate_deploy" {
  filename      = "/tmp/lambda_validate_deploy.rb.zip"
  function_name = "RetrievalValidateBeforeTraffic"
  role          = aws_iam_role.lambda_validate_deploy.arn
  handler       = "lambda_validate_deploy.handler"

  source_code_hash = data.archive_file.lambda_validate_deploy.output_base64sha256

  runtime = "ruby2.7"

  environment {
    variables = {
      TEST_LB_ENDPOINT         = "https://${aws_lb.covidshield_key_retrieval.dns_name}:8443/services/ping"
      CLOUDFRONT_CUSTOM_HEADER = var.cloudfront_custom_header
    }
  }

}

resource "aws_lambda_function" "submission_lambda_validate_deploy" {
  filename      = "/tmp/lambda_validate_deploy.rb.zip"
  function_name = "SubmissionValidateBeforeTraffic"
  role          = aws_iam_role.lambda_validate_deploy.arn
  handler       = "lambda_validate_deploy.handler"

  source_code_hash = data.archive_file.lambda_validate_deploy.output_base64sha256

  runtime = "ruby2.7"

  environment {
    variables = {
      TEST_LB_ENDPOINT = "https://${aws_lb.covidshield_key_submission.dns_name}:8443/services/ping"
    }
  }

}
