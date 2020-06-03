locals {
  image_names = ["key-retrieval", "key-submission", "monolith"]
}

resource "aws_ecr_repository" "repository" {
  for_each             = ${local.image_names}
  name                 = "covid-server/${each.value}"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "policy" {
  for_each   = ${local.image_names}
  repository = aws_ecr_repository.repository[each.value].name

  policy = <<EOF
{
    "rules": [
        {
            "rulePriority": 1,
            "description": "Keep last 30 images",
            "selection": {
                "tagStatus": "tagged",
                "tagPrefixList": ["v"],
                "countType": "imageCountMoreThan",
                "countNumber": 30
            },
            "action": {
                "type": "expire"
            }
        }
    ]
}
EOF
}
