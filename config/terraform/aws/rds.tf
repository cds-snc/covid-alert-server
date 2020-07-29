resource "random_string" "random" {
  length  = 6
  special = false
  upper   = false
}

resource "aws_db_subnet_group" "covidshield" {
  name       = var.rds_db_subnet_group_name
  subnet_ids = aws_subnet.covidshield_private.*.id

  tags = {
    Name                  = var.rds_db_subnet_group_name
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_rds_cluster_instance" "covidshield_server_instances" {
  count                        = 3
  identifier                   = "${var.rds_server_db_name}-instance-${count.index}"
  cluster_identifier           = aws_rds_cluster.covidshield_server.id
  instance_class               = var.rds_server_instance_class
  db_subnet_group_name         = aws_db_subnet_group.covidshield.name
  performance_insights_enabled = true

  tags = {
    Name                  = "${var.rds_server_db_name}-instance"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_rds_cluster" "covidshield_server" {
  cluster_identifier        = "${var.rds_server_db_name}-cluster"
  engine                    = "aurora"
  database_name             = var.rds_server_db_name
  final_snapshot_identifier = "server-${random_string.random.result}"
  master_username           = var.rds_server_db_user
  master_password           = var.rds_server_db_password
  backup_retention_period   = 1
  preferred_backup_window   = "07:00-09:00"
  db_subnet_group_name      = aws_db_subnet_group.covidshield.name
  storage_encrypted         = true


  vpc_security_group_ids = [
    aws_security_group.covidshield_database.id
  ]

  tags = {
    Name                  = "${var.rds_server_db_name}-cluster"
    (var.billing_tag_key) = var.billing_tag_value
  }
}
