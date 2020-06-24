###
# AWS VPC
###

data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "covidshield" {
  cidr_block = var.vpc_cidr_block

  tags = {
    Name                  = var.vpc_name
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS Internet Gateway
###

resource "aws_internet_gateway" "covidshield" {
  vpc_id = aws_vpc.covidshield.id

  tags = {
    Name                  = var.vpc_name
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS Subnets
###

resource "aws_subnet" "covidshield_private" {
  count = 3

  vpc_id            = aws_vpc.covidshield.id
  cidr_block        = cidrsubnet(var.vpc_cidr_block, 8, count.index)
  availability_zone = element(data.aws_availability_zones.available.names, count.index)

  tags = {
    Name                  = "Private Subnet 0${count.index + 1}"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_subnet" "covidshield_public" {
  count = 3

  vpc_id            = aws_vpc.covidshield.id
  cidr_block        = cidrsubnet(var.vpc_cidr_block, 8, count.index + 3)
  availability_zone = element(data.aws_availability_zones.available.names, count.index)

  tags = {
    Name                  = "Public Subnet 0${count.index + 1}"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS NAT GW
###

resource "aws_eip" "covidshield_natgw" {
  depends_on = [aws_internet_gateway.covidshield]

  vpc = true

  tags = {
    Name                  = "${var.vpc_name} NAT GW"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_nat_gateway" "covidshield" {
  depends_on = [aws_internet_gateway.covidshield]

  allocation_id = aws_eip.covidshield_natgw.id
  subnet_id     = aws_subnet.covidshield_public.0.id

  tags = {
    Name                  = "${var.vpc_name} NAT GW"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS Routes
###

resource "aws_default_route_table" "covidshield" {
  default_route_table_id = aws_vpc.covidshield.default_route_table_id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.covidshield.id
  }

  tags = {
    Name                  = "Default Route Table"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_route_table" "covidshield_public_subnet" {
  vpc_id = aws_vpc.covidshield.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.covidshield.id
  }


  tags = {
    Name                  = "Public Subnet Route Table"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_route_table_association" "covidshield" {
  count = 3

  subnet_id      = aws_subnet.covidshield_public.*.id[count.index]
  route_table_id = aws_route_table.covidshield_public_subnet.id
}


###
# AWS Security Groups
###

resource "aws_security_group" "covidshield_key_retrieval" {
  name        = "covidshield-key-retrieval"
  description = "Ingress - CovidShield Key Retrieval App"
  vpc_id      = aws_vpc.covidshield.id

  ingress {
    protocol  = "tcp"
    from_port = 8001
    to_port   = 8001
    security_groups = [
      aws_security_group.covidshield_load_balancer.id
    ]
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_security_group" "covidshield_key_submission" {
  name        = "covidshield-key-submission"
  description = "Ingress - CovidShield Key Submission App"
  vpc_id      = aws_vpc.covidshield.id

  ingress {
    protocol  = "tcp"
    from_port = 8000
    to_port   = 8000
    security_groups = [
      aws_security_group.covidshield_load_balancer.id
    ]
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_security_group" "covidshield_load_balancer" {
  name        = "covidshield-load-balancer"
  description = "Ingress - CovidShield Load Balancer"
  vpc_id      = aws_vpc.covidshield.id

  ingress {
    protocol    = "tcp"
    from_port   = 443
    to_port     = 443
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    protocol    = "tcp"
    from_port   = 8001
    to_port     = 8001
    cidr_blocks = [var.vpc_cidr_block]
  }

  egress {
    protocol    = "tcp"
    from_port   = 8000
    to_port     = 8000
    cidr_blocks = [var.vpc_cidr_block]
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_security_group" "covidshield_database" {
  name        = "covidshield-database"
  description = "Ingress - CovidShield Database"
  vpc_id      = aws_vpc.covidshield.id

  ingress {
    protocol  = "tcp"
    from_port = 3306
    to_port   = 3306
    security_groups = [
      aws_security_group.covidshield_key_retrieval.id,
      aws_security_group.covidshield_key_submission.id
    ]
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_security_group" "covidshield_egress_anywhere" {
  name        = "egress-anywhere"
  description = "Egress - CovidShield Anywhere"
  vpc_id      = aws_vpc.covidshield.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    (var.billing_tag_key) = var.billing_tag_value
  }
}
