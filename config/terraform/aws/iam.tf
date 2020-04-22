data "aws_iam_policy_document" "covidshield" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

###
# AWS IAM - Key Retrieval
###

data "aws_iam_policy_document" "covidshield_secrets_manager_key_retrieval" {
  statement {
    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      aws_secretsmanager_secret.key_retrieval_env_hmac_key.arn,
      aws_secretsmanager_secret.backend_database_url.arn,
    ]
  }
}

resource "aws_iam_policy" "covidshield_secrets_manager_key_retrieval" {
  name   = "CovidShieldSecretsManagerKeyRetrieval"
  path   = "/"
  policy = data.aws_iam_policy_document.covidshield_secrets_manager_key_retrieval.json
}

resource "aws_iam_role" "covidshield_key_retrieval" {
  name = var.ecs_key_retrieval_name

  assume_role_policy = data.aws_iam_policy_document.covidshield.json

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_key_retrieval" {
  role       = aws_iam_role.covidshield_key_retrieval.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "secrets_manager_key_retrieval" {
  role       = aws_iam_role.covidshield_key_retrieval.name
  policy_arn = aws_iam_policy.covidshield_secrets_manager_key_retrieval.arn
}


###
# AWS IAM - Key Submission
###

data "aws_iam_policy_document" "covidshield_secrets_manager_key_submission" {
  statement {
    effect = "Allow"

    actions = [
      "secretsmanager:GetSecretValue",
    ]

    resources = [
      aws_secretsmanager_secret.key_submission_env_key_claim_token.arn,
      aws_secretsmanager_secret.backend_database_url.arn,
    ]
  }
}

resource "aws_iam_policy" "covidshield_secrets_manager_key_submission" {
  name   = "CovidShieldSecretsManagerKeySubmission"
  path   = "/"
  policy = data.aws_iam_policy_document.covidshield_secrets_manager_key_submission.json
}

resource "aws_iam_role" "covidshield_key_submission" {
  name = var.ecs_key_submission_name

  assume_role_policy = data.aws_iam_policy_document.covidshield.json

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_key_submission" {
  role       = aws_iam_role.covidshield_key_submission.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy_attachment" "secrets_manager_key_submission" {
  role       = aws_iam_role.covidshield_key_submission.name
  policy_arn = aws_iam_policy.covidshield_secrets_manager_key_submission.arn
}
