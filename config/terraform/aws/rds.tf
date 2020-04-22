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

resource "aws_db_instance" "covidshield_backend" {
  identifier_prefix         = "backend"
  allocated_storage         = var.rds_backend_allocated_storage
  storage_type              = "gp2"
  engine                    = "mysql"
  engine_version            = "5.7"
  final_snapshot_identifier = "backend-${random_string.random.result}"
  skip_final_snapshot       = false
  instance_class            = var.rds_backend_instance_class
  name                      = var.rds_backend_db_name
  username                  = var.rds_backend_db_user
  password                  = var.rds_backend_db_password
  vpc_security_group_ids = [
    aws_security_group.covidshield_database.id
  ]
  parameter_group_name = "default.mysql5.7"
  db_subnet_group_name = aws_db_subnet_group.covidshield.id

  tags = {
    Name                  = var.rds_backend_db_name
    (var.billing_tag_key) = var.billing_tag_value
  }
}
