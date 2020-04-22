resource "aws_secretsmanager_secret" "backend_database_url" {
  name = "backend-database-url"
}

resource "aws_secretsmanager_secret_version" "backend_database_url" {
  secret_id     = aws_secretsmanager_secret.backend_database_url.id
  secret_string = "${var.rds_backend_db_user}:${var.rds_backend_db_password}@tcp(${aws_db_instance.covidshield_backend.endpoint})/${var.rds_backend_db_name}"
}

###
# AWS Secret Manager - Key Retrieval
###

resource "aws_secretsmanager_secret" "key_retrieval_env_hmac_key" {
  name = "key-retrieval-env-hmac-key"
}

resource "aws_secretsmanager_secret_version" "key_retrieval_env_hmac_key" {
  secret_id     = aws_secretsmanager_secret.key_retrieval_env_hmac_key.id
  secret_string = var.ecs_task_key_retrieval_env_hmac_key
}

###
# AWS Secret Manager - Key Submission
###

resource "aws_secretsmanager_secret" "key_submission_env_key_claim_token" {
  name = "key-submission-env-key-claim-token"
}

resource "aws_secretsmanager_secret_version" "key_submission_env_key_claim_token" {
  secret_id     = aws_secretsmanager_secret.key_submission_env_key_claim_token.id
  secret_string = var.ecs_task_key_submission_env_key_claim_token
}
