resource "aws_secretsmanager_secret" "server_database_url" {
  name = "server-database-url"
}

resource "aws_secretsmanager_secret_version" "server_database_url" {
  secret_id     = aws_secretsmanager_secret.server_database_url.id
  secret_string = "${var.rds_server_db_user}:${var.rds_server_db_password}@tcp(${aws_db_instance.covidshield_server.endpoint})/${var.rds_server_db_name}"
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

resource "aws_secretsmanager_secret" "key_retrieval_env_ecdsa_key" {
  name = "key-retrieval-env-ecdsa-key"
}

resource "aws_secretsmanager_secret_version" "key_retrieval_env_ecdsa_key" {
  secret_id     = aws_secretsmanager_secret.key_retrieval_env_ecdsa_key.id
  secret_string = var.ecs_task_key_retrieval_env_ecdsa_key
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
