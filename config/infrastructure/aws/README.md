# Deploying Covid Shield on Amazon Web Services (AWS)

:warning: This is not a fully featured production environment and aims to provide an accessible overview of the service.

This document describes how to deploy and operate a **reference implementation** of the Covid Shield web portal, along with the diagnosis key retrieval and submission services on [AWS](https://aws.amazon.com/).

There should be an illustration of the Covid Shield infrastructure deployed on AWS right here

At a glance, health care professionals (on the left) interact with a web portal, and mobile app users (on the right) interact with the diagnosis key retrieval and submission services.

## IT Service Requirements

| Service | AWS product offering |
|---------|---------|
| Serverless compute | [Fargate](https://aws.amazon.com/fargate/) |
| Container registry | [Elastic Container Registry](https://aws.amazon.com/ecr/) |
| Domain name services | [Route 53](https://aws.amazon.com/route53/) |
| TLS certificates | [Certificate Manager](https://aws.amazon.com/certificate-manager/) |
| Load balancing | [Elastic Load Balancing](https://aws.amazon.com/elasticloadbalancing/) |
| Content delivery network | [CloudFront](https://aws.amazon.com/cloudfront/) |
| Web application firewall | [WAF](https://aws.amazon.com/waf/) |

## Deploying Covid Shield :shield:

While this infrastructure may be deployed in a number of different ways, this document demonstrates a deploy using a small series of command line operations to generate credentials, and a CI/CD pipeline using [GitHub Actions](https://github.com/features/actions), [Docker](https://www.docker.com/why-docker), and [Terraform](https://www.terraform.io/).

### Prerequisites

- A [GitHub repository](https://help.github.com/en/github/getting-started-with-github/create-a-repo) with [Actions](https://github.com/features/actions) enabled

- [`aws` Command Line Interface](https://aws.amazon.com/cli/) installed and available in your path

- [`terraform`](https://www.terraform.io/downloads.html) 0.12.x installed and available in your path

#### Optional TODO: make a quick playbook doc to see if the containers are up or down? or pivot to the ui?

- [`ecs-cli`](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ECS_CLI.html) installed and available in your path

## Deploying to AWS with Terraform

The credentials for the AWS Terraform provider are expected to be provided through the standard AWS credential environment variables.

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY_ID`

If it's not already done, and before applying. You will need to enable the new ARN format to propagate tags to containers.

```
aws ecs put-account-setting-default --name serviceLongArnFormat --value enabled
aws ecs put-account-setting-default --name taskLongArnFormat --value enabled
aws ecs put-account-setting-default --name containerInstanceLongArnFormat --value enabled
```

All Terraform variables are defined in `config/terraform/aws/variables.tf` & their values are set in `config/terraform/aws/variables.auto.tfvars`. There are **four** secret variables that should be set through the following environment variables as to not commit plain text secrets to version control.

- `TF_VAR_ecs_task_key_retrieval_env_ecdsa_key`
- `TF_VAR_ecs_task_key_retrieval_env_hmac_key`
- `TF_VAR_ecs_task_key_submission_env_key_claim_token`
- `TF_VAR_rds_backend_db_password`

If you are using Terraform in Github actions the above can be set as Github secrets, and set as environment variables in your YAML file (see `.github/workflows/terraform.yml`).

There is an optional Terraform variable that can be set to control which container to deploy. It should match a container tag that both Key Retrieval & Key Submission share. By default Terraform will deploy the latest commit on the master branch.

- `TF_VAR_github_sha`

To run manually:
1. Go to the AWS Terraform directory - `cd config/terraform/aws`
2. Run
> TF_VAR_ecs_task_key_retrieval_env_ecdsa_key="******" TF_VAR_ecs_task_key_retrieval_env_hmac_key="******" TF_VAR_ecs_task_key_submission_env_key_claim_token="******" TF_VAR_rds_backend_db_password="******" AWS_ACCESS_KEY_ID="******" AWS_SECRET_ACCESS_KEY="******" terraform [init|plan|apply]

## Building and releasing applications with GitHub and Docker :whale:

This section will demonstrate the following:

- Continuous container builds on [new pull requests](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-requests)
- Releasing new containers on a [successful merge](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/merging-a-pull-request) to the master branch
- Integrating dependency management using [Dependabot](https://dependabot.com/)

### Getting started with GitHub Actions

According to GitHub, [Actions](https://github.com/features/actions) makes it easy to automate all your software workflows, now with world-class CI/CD. Build, test, and deploy your code right from GitHub. Make code reviews, branch management, and issue triaging work the way you want.

We've chosen to configure a number of Actions to make the infrastructure go brrrr :robot:, here they are:

1. [ci](to test a thing), for container vulnerability scanning
2. [ci](to scan a thing), for __
3. [ci](docker build), for __
4. [ci](docker push), for __
5. [cd](to ecs), for __

#### Action #1: test a thing

#### Action #2: scan a thing

#### Action #3: build a thing

#### Action #4: push a thing

#### Action #5: deploy a thing

### Integrating dependency management with Dependabot

According to GitHub (because they [own it](https://dependabot.com/blog/hello-github/)), Dependabot creates pull requests to keep your dependencies secure and up-to-date.

Configuring Dependabot requires one or two things:

1. [Sign up](https://app.dependabot.com/auth/sign-up) and grant Dependabot access to your repositories
2. You can configure the repository in the UI after signing up. Alternatively, you may specify a [Dependabot config file](https://dependabot.com/docs/config-file/) in your repository at `.dependabot/config.yml`

Here is the Dependabot configuration we chose for the [Portal](https://github.com/CovidShield/portal):

```yaml
version: 1
update_configs:
- package_manager: "ruby:bundler"
  directory: "/"
  update_schedule: "weekly"
- package_manager: "javascript"
  directory: "/"
  update_schedule: "weekly"
```

This configuration schedules the :robot: to [check for updates](https://dependabot.com/#how-it-works) every week for both the [Bundler](https://bundler.io/) and [Yarn](https://yarnpkg.com/) packages deployed with this application. Feel free to run this configurations through the [configuration validator](https://dependabot.com/docs/config-file/validator/) to attest that it is valid.
