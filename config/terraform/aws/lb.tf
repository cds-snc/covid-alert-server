###
# AWS LB - Key Retrieval Target Groups
###

resource "aws_lb_target_group" "covidshield_key_retrieval_green" {
  name                 = "covidshield-key-retrieval-green"
  port                 = 8001
  protocol             = "HTTP"
  target_type          = "ip"
  deregistration_delay = 30
  vpc_id               = aws_vpc.covidshield.id

  health_check {
    enabled             = true
    interval            = 10
    path                = "/services/ping"
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    Name                  = "covidshield-key-retrieval"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb_target_group" "covidshield_key_retrieval_blue" {
  name                 = "covidshield-key-retrieval-blue"
  port                 = 8001
  protocol             = "HTTP"
  target_type          = "ip"
  deregistration_delay = 30
  vpc_id               = aws_vpc.covidshield.id

  health_check {
    enabled             = true
    interval            = 10
    path                = "/services/ping"
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    Name                  = "covidshield-key-retrieval"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

###
# AWS LB - Key Submission Target Groups
###

resource "aws_lb_target_group" "covidshield_key_submission_blue" {
  name                 = "covidshield-key-submission-blue"
  port                 = 8000
  protocol             = "HTTP"
  target_type          = "ip"
  deregistration_delay = 30
  vpc_id               = aws_vpc.covidshield.id

  health_check {
    enabled             = true
    interval            = 10
    path                = "/services/ping"
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    Name                  = "covidshield-key-submission"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb_target_group" "covidshield_key_submission_green" {
  name                 = "covidshield-key-submission-green"
  port                 = 8000
  protocol             = "HTTP"
  target_type          = "ip"
  deregistration_delay = 30
  vpc_id               = aws_vpc.covidshield.id

  health_check {
    enabled             = true
    interval            = 10
    path                = "/services/ping"
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }

  tags = {
    Name                  = "covidshield-key-submission"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb" "covidshield_key_server" {
  name               = "covidshield-key-server"
  internal           = false
  load_balancer_type = "application"
  security_groups = [
    aws_security_group.covidshield_load_balancer.id
  ]
  subnets = aws_subnet.covidshield_public.*.id

  tags = {
    Name                  = "covidshield-key-server"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb_listener" "covidshield_key_server" {
  depends_on = [
    aws_acm_certificate.submission_covidshield,
    aws_route53_record.submission_covidshield_certificate_validation,
    aws_acm_certificate_validation.submission_covidshield,
  ]

  load_balancer_arn = aws_lb.covidshield_key_server.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate.submission_covidshield.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.covidshield_key_retrieval_green.arn
  }

  lifecycle {
    ignore_changes = [
      default_action # updated by codedeploy
    ]
  }

}

resource "aws_lb_listener_rule" "static" {
  listener_arn = aws_lb_listener.covidshield_key_server.arn
  priority     = 100

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.covidshield_key_submission_blue.arn
  }

  condition {
    path_pattern {
      values = [
        "/new-key-claim*",
        "/claim-key",
        "/upload",
        "/exposure-configuration*",
        "/services*"
      ]
    }
  }
}
