###
# AWS LB - Key Retrieval
###

resource "aws_lb_target_group" "covidshield_key_retrieval" {
  name                 = "covidshield-key-retrieval"
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

resource "aws_lb_target_group" "covidshield_key_retrieval_2" {
  name                 = "covidshield-key-retrieval-2"
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
    Name                  = "covidshield-key-retrieval-2"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb" "covidshield_key_retrieval" {
  name               = "covidshield-key-retrieval"
  internal           = false
  load_balancer_type = "application"
  security_groups = [
    aws_security_group.covidshield_load_balancer.id
  ]
  subnets = aws_subnet.covidshield_public.*.id

  tags = {
    Name                  = "covidshield-key-retrieval"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb_listener" "covidshield_key_retrieval" {
  depends_on = [
    aws_acm_certificate.covidshield,
    aws_route53_record.covidshield_certificate_validation,
    aws_acm_certificate_validation.covidshield,
  ]

  load_balancer_arn = aws_lb.covidshield_key_retrieval.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate.covidshield.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.covidshield_key_retrieval.arn
  }

  lifecycle {
    ignore_changes = [
      default_action # updated by codedeploy
    ]
  }

}

###
# AWS LB - Key Submission
###

resource "aws_lb_target_group" "covidshield_key_submission" {
  name                 = "covidshield-key-submission"
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

resource "aws_lb_target_group" "covidshield_key_submission_2" {
  name                 = "covidshield-key-submission-2"
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
    Name                  = "covidshield-key-submission-2"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb" "covidshield_key_submission" {
  name               = "covidshield-key-submission"
  internal           = false
  load_balancer_type = "application"
  security_groups = [
    aws_security_group.covidshield_load_balancer.id
  ]
  subnets = aws_subnet.covidshield_public.*.id

  tags = {
    Name                  = "covidshield-key-submission"
    (var.billing_tag_key) = var.billing_tag_value
  }
}

resource "aws_lb_listener" "covidshield_key_submission" {
  depends_on = [
    aws_acm_certificate.covidshield,
    aws_route53_record.covidshield_certificate_validation,
    aws_acm_certificate_validation.covidshield,
  ]

  load_balancer_arn = aws_lb.covidshield_key_submission.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2019-08"
  certificate_arn   = aws_acm_certificate.covidshield.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.covidshield_key_submission.arn
  }

  lifecycle {
    ignore_changes = [
      default_action # updated by codedeploy
    ]
  }

}
