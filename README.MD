# Product API on AWS ECS

A Go + Gin implementation of the Product endpoints from `api.yaml`, deployed to AWS ECS (Fargate) with Terraform. Products are stored in memory, so they are lost whenever the task restarts.

| Method | Path | Responses |
|---|---|---|
| `GET` | `/products/{productId}` | `200`, `404`, `500` |
| `POST` | `/products/{productId}/details` | `204`, `400`, `404`, `500` |

## What gets deployed

Running `terraform apply` from `terraform/` does all of this:

1. Creates an ECR repository.
2. Builds the Docker image from `src/` and pushes it to ECR.
3. Creates a CloudWatch log group, a security group that opens port `8080`, an ECS cluster, a task definition and an ECS service.
4. Starts one Fargate task with a public IP in the default VPC.

```
.
├── src/                    # Product API (Go)
│   ├── product-service.go
│   ├── go.mod / go.sum
│   └── Dockerfile
└── terraform/
    ├── main.tf             # wires the modules together, builds and pushes the image
    ├── variables.tf        # region, names, port, task count
    ├── provider.tf         # AWS + Docker providers
    ├── outputs.tf
    └── modules/            # network, ecr, logging, ecs
```

## Prerequisites

Install these on your machine:

| Tool | Check it's installed | Install (macOS) |
|---|---|---|
| Terraform 1.5+ | `terraform -version` | `brew tap hashicorp/tap && brew install hashicorp/tap/terraform` |
| AWS CLI v2 | `aws --version` | `brew install awscli` |
| Docker Desktop | `docker info` | [docker.com](https://www.docker.com/products/docker-desktop/) |
| Git | `git --version` | `xcode-select --install` |

You also need an **AWS Academy Learner Lab** session. The Terraform reuses the lab's `LabRole` IAM role, so it won't work in a regular AWS account without changes.

Docker Desktop must be **running** during `terraform apply`, because Terraform builds the image locally.

## Deploy

### 1. Get the code

```bash
git clone https://github.com/ameir-elouadi/CS6650_2b_demo.git
cd CS6650_2b_demo
```

### 2. Set your AWS credentials

In Learner Lab, click **Start Lab**, wait for the dot to turn green, then open **AWS Details** → **AWS CLI: Show**. It shows `aws_access_key_id`, `aws_secret_access_key` and `aws_session_token`.

```bash
aws configure
```

Enter the access key ID and secret access key. For the region, enter `us-west-2` (the region the Terraform deploys to). For output format, enter `json`. Then set the session token:

```bash
aws configure set aws_session_token <YOUR-SESSION-TOKEN>
```

Check that it worked. This should print your account details:

```bash
aws sts get-caller-identity
```

Learner Lab credentials expire when the session ends (about 4 hours). After restarting the lab, repeat this step with the new values.

### 3. Apply the infrastructure

```bash
cd terraform
terraform init
terraform apply -auto-approve
```

The first run takes a few minutes, mostly for the image build and push. It ends with:

```
Outputs:

aws_region = "us-west-2"
ecs_cluster_name = "CS6650L2-cluster"
ecs_service_name = "CS6650L2"
```

### 4. Get the server's public IP

Fargate gives the task a new public IP each time it starts, so look it up after every deploy. From the `terraform/` folder:

```bash
REGION=$(terraform output -raw aws_region)
CLUSTER=$(terraform output -raw ecs_cluster_name)
SERVICE=$(terraform output -raw ecs_service_name)

TASK=$(aws ecs list-tasks --region $REGION --cluster $CLUSTER --service-name $SERVICE \
  --query 'taskArns[0]' --output text)

ENI=$(aws ecs describe-tasks --region $REGION --cluster $CLUSTER --tasks $TASK \
  --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" --output text)

IP=$(aws ec2 describe-network-interfaces --region $REGION --network-interface-ids $ENI \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "API: http://$IP:8080"
```

If `TASK` comes back as `None`, the task hasn't started yet. Wait 30–60 seconds and run the commands again.

## Send requests

Run these from the same terminal, so `$IP` is set. Send them in order: the `200` example needs the product created by the `204` example.

### POST /products/{productId}/details

**204 No Content: product created** (or replaced, if it already exists)
```bash
curl -i -X POST http://$IP:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 204 No Content
```

**400 Bad Request: invalid input**

All six fields are required and must follow the limits in `api.yaml`. The `product_id` in the body must match the one in the URL.
```bash
curl -i -X POST http://$IP:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345}'
```
```
HTTP/1.1 400 Bad Request

{"error":"INVALID_INPUT","message":"The provided input data is invalid","details":"Key: 'Product.SKU' Error:Field validation for 'SKU' failed on the 'required' tag ..."}
```

**404 Not Found: invalid product ID**

A product ID that isn't a positive 32-bit integer (e.g. `abc` or `0`) can't identify a product.
```bash
curl -i -X POST http://$IP:8080/products/abc/details \
  -H "Content-Type: application/json" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 404 Not Found

{"error":"NOT_FOUND","message":"Product not found","details":"product ID must be a positive 32-bit integer"}
```

**500 Internal Server Error: simulated failure**

Any request with the header `X-Simulate-Error: true` returns `500`, so the error response can be tested.
```bash
curl -i -X POST http://$IP:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -H "X-Simulate-Error: true" \
  -d '{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}'
```
```
HTTP/1.1 500 Internal Server Error

{"error":"INTERNAL_ERROR","message":"Internal server error"}
```

### GET /products/{productId}

**200 OK: product found**
```bash
curl -i http://$IP:8080/products/12345
```
```
HTTP/1.1 200 OK

{"product_id":12345,"sku":"ABC-123-XYZ","manufacturer":"Acme Corporation","category_id":456,"weight":1250,"some_other_id":789}
```

**404 Not Found: product doesn't exist**
```bash
curl -i http://$IP:8080/products/99999
```
```
HTTP/1.1 404 Not Found

{"error":"NOT_FOUND","message":"Product not found"}
```

**500 Internal Server Error: simulated failure**
```bash
curl -i http://$IP:8080/products/12345 -H "X-Simulate-Error: true"
```
```
HTTP/1.1 500 Internal Server Error

{"error":"INTERNAL_ERROR","message":"Internal server error"}
```

## Deploying code changes

The service always runs the `latest` image tag, so after editing code in `src/`, rebuild the image and restart the task:

```bash
cd terraform
terraform apply -auto-approve -replace=docker_image.app -replace=docker_registry_image.app
aws ecs update-service --region $(terraform output -raw aws_region) \
  --cluster $(terraform output -raw ecs_cluster_name) \
  --service $(terraform output -raw ecs_service_name) \
  --force-new-deployment > /dev/null
```

The new task gets a new IP, so repeat [step 4](#4-get-the-servers-public-ip) after a minute.

## Run locally (optional)

```bash
cd src
go mod tidy
go run product-service.go
```

Or with Docker:

```bash
cd src
docker build -t product-api .
docker run --rm -p 8080:8080 product-api
```

Then use `http://localhost:8080` in place of `http://$IP:8080` in the examples above.

## Clean up

When you're done, delete everything so it doesn't keep using lab credits. From `terraform/`:

```bash
terraform destroy -auto-approve
```

## Troubleshooting

| Problem | Fix |
|---|---|
| `ExpiredToken` or `InvalidClientTokenId` | The lab session ended. Restart it and redo [step 2](#2-set-your-aws-credentials). |
| `Cannot connect to the Docker daemon` | Start Docker Desktop and wait until `docker info` works. If it still fails on macOS, run `export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock` and apply again. |
| `curl` times out | The task may still be starting, or it restarted with a new IP. Wait a minute and redo step 4. |
| Task keeps stopping | Check the logs: `aws logs tail /ecs/CS6650L2 --region us-west-2 --since 15m` |
| `RepositoryAlreadyExistsException` or `ResourceAlreadyExists` | A previous deploy in this lab account wasn't destroyed. Run `terraform destroy` from the same folder, or delete the leftover resources in the AWS console. |

The Docker image is built for `linux/amd64`, the architecture Fargate uses by default, so it deploys correctly from both Intel and Apple Silicon Macs.
