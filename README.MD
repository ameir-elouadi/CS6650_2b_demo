# Product API on AWS ECS

A Go + Gin implementation of the Product endpoints from the course's `api.yaml`, deployed to AWS ECS (Fargate) with Terraform. Products are stored in memory, so they are lost whenever the task restarts.

| Method | Path | Responses |
|---|---|---|
| `GET` | `/products/{productId}` | `200`, `404`, `500` |
| `POST` | `/products/{productId}/details` | `204`, `400`, `404`, `500` |

There are two ways to use this repo:

- **Deploy your own copy** to your own AWS Learner Lab account: follow [Deploy from your machine](#deploy-from-your-machine), then [Send requests](#send-requests).
- **Send requests to someone else's deployment**: ask them for their server's IP, then go straight to [Send requests](#send-requests). The IP changes every time their task restarts, so check it's current.

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

## Deploy from your machine

### 1. Install the prerequisites

| Tool | Check it's installed | Install (macOS) |
|---|---|---|
| Terraform 1.5+ | `terraform -version` | `brew tap hashicorp/tap && brew install hashicorp/tap/terraform` |
| AWS CLI v2 | `aws --version` | `brew install awscli` |
| Docker Desktop | `docker info` | [docker.com](https://www.docker.com/products/docker-desktop/) |
| Git | `git --version` | `xcode-select --install` |

You also need your own **AWS Academy Learner Lab** session. The Terraform reuses the lab's `LabRole` IAM role, so it won't work in a regular AWS account without changes. Each person deploys into their own lab account, so group members don't conflict with each other.

### 2. Get the code

```bash
git clone https://github.com/ameir-elouadi/CS6650_2b_demo.git
cd CS6650_2b_demo
```

### 3. Set your AWS credentials

In Learner Lab, click **Start Lab** and wait for the dot to turn green. Then open **AWS Details** → **AWS CLI: Show**. It shows a block like this:

```
[default]
aws_access_key_id=...
aws_secret_access_key=...
aws_session_token=...
```

Copy the whole block into `~/.aws/credentials`, replacing anything already there. To open the file:

```bash
mkdir -p ~/.aws && open -e ~/.aws/credentials
```

If the file doesn't exist yet, run `touch ~/.aws/credentials` first. Then set the region:

```bash
aws configure set region us-west-2
```

Check that it worked. This should print your account details, not an error:

```bash
aws sts get-caller-identity
```

Learner Lab credentials expire when the session ends (about 4 hours). After restarting the lab, paste the new block into `~/.aws/credentials` again.

### 4. Start Docker

Open Docker Desktop and wait until `docker info` runs without an error. Terraform builds the image on your machine, so Docker must be running during `terraform apply`.

On macOS, also point Terraform at Docker Desktop's socket. Run this in the same terminal you'll run Terraform from:

```bash
export DOCKER_HOST=unix://$HOME/.docker/run/docker.sock
```

Without it, Terraform fails with `Cannot connect to the Docker daemon at unix:///var/run/docker.sock`.

### 5. Deploy

From the repo root:

```bash
cd terraform
terraform init
terraform apply
```

Terraform lists what it will create (`Plan: 8 to add, 0 to change, 0 to destroy`) and asks for confirmation. Type `yes`. The first run takes a few minutes, mostly for the image build and push. It ends with:

```
Apply complete! Resources: 8 added, 0 changed, 0 destroyed.

Outputs:

aws_region = "us-west-2"
ecs_cluster_name = "CS6650L2-cluster"
ecs_service_name = "CS6650L2"
```

### 6. Get the server's public IP

Fargate gives the task a new public IP each time it starts, so look it up after every deploy. Wait about a minute after `terraform apply` finishes, then run this from the `terraform/` folder:

```bash
REGION=$(terraform output -raw aws_region)
CLUSTER=$(terraform output -raw ecs_cluster_name)
SERVICE=$(terraform output -raw ecs_service_name)

TASK=$(aws ecs list-tasks --region $REGION \
  --cluster $CLUSTER --service-name $SERVICE \
  --query 'taskArns[0]' --output text)

ENI=$(aws ecs describe-tasks --region $REGION \
  --cluster $CLUSTER --tasks $TASK \
  --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value" \
  --output text)

IP=$(aws ec2 describe-network-interfaces --region $REGION \
  --network-interface-ids $ENI \
  --query 'NetworkInterfaces[0].Association.PublicIp' --output text)

echo "API: http://$IP:8080"
```

If `TASK` comes back as `None`, or the IP is empty, the task hasn't started yet. Wait 30–60 seconds and run the commands again.

You can also find the IP in the AWS console: **ECS** → **Clusters** → `CS6650L2-cluster` → **Tasks** → click the running task → **Public IP**.

## Send requests

### Set the IP

If you just ran step 6, `$IP` is already set in that terminal. Otherwise, set it to the server's IP:

```bash
IP=<SERVER-IP>
```

Every command below uses `$IP`, so run them all in the same terminal window.

### Run the requests

Paste each block on its own, in this order: the `200` GET needs the product created by the `204` POST. `-i` makes curl print the status line first, e.g. `HTTP/1.1 204 No Content`.

The JSON bodies are split across short lines on purpose. If a long one-line command wraps when you copy it, a line break can land inside a string like `"Acme Corporation"`, and the API rejects it with `400 invalid character '\n' in string literal`.

#### POST /products/{productId}/details

**204 No Content: product created** (or replaced, if it already exists)
```bash
curl -i -X POST http://$IP:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{
    "product_id": 12345,
    "sku": "ABC-123-XYZ",
    "manufacturer": "Acme Corporation",
    "category_id": 456,
    "weight": 1250,
    "some_other_id": 789
  }'
```
```
HTTP/1.1 204 No Content
```

**400 Bad Request: invalid input**

All six fields are required and must follow the limits in `api.yaml`. The `product_id` in the body must match the one in the URL.
```bash
curl -i -X POST http://$IP:8080/products/12345/details \
  -H "Content-Type: application/json" \
  -d '{"product_id": 12345}'
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
  -d '{
    "product_id": 12345,
    "sku": "ABC-123-XYZ",
    "manufacturer": "Acme Corporation",
    "category_id": 456,
    "weight": 1250,
    "some_other_id": 789
  }'
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
  -d '{
    "product_id": 12345,
    "sku": "ABC-123-XYZ",
    "manufacturer": "Acme Corporation",
    "category_id": 456,
    "weight": 1250,
    "some_other_id": 789
  }'
```
```
HTTP/1.1 500 Internal Server Error

{"error":"INTERNAL_ERROR","message":"Internal server error"}
```

#### GET /products/{productId}

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

In zsh, a `%` at the end of a response just means the body didn't end with a newline. It isn't part of the response.

## Check the logs

Every request is logged to CloudWatch. From any folder:

```bash
aws logs tail /ecs/CS6650L2 --region us-west-2 --since 15m
```

Each request shows up as a `[GIN]` line with its status code. The simulated `500` works by panicking, so those requests also print a Go stack trace. That's expected: the server recovers and keeps running.

## Deploying code changes

The service always runs the `latest` image tag, so after editing code in `src/`, rebuild the image and restart the task. From `terraform/`, with Docker running and `DOCKER_HOST` set (step 4):

```bash
terraform apply -replace=docker_image.app -replace=docker_registry_image.app
aws ecs update-service --region $(terraform output -raw aws_region) \
  --cluster $(terraform output -raw ecs_cluster_name) \
  --service $(terraform output -raw ecs_service_name) \
  --force-new-deployment > /dev/null
```

The new task gets a new IP, so repeat [step 6](#6-get-the-servers-public-ip) after a minute.

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

Then set `IP=localhost` and use the requests above.

## Clean up

When you're done, delete everything so it doesn't keep using lab credits. From `terraform/`:

```bash
terraform destroy
```

Type `yes` to confirm. Run it from the same machine and folder you deployed from, since that's where Terraform keeps its state.

## Troubleshooting

| Problem | Fix |
|---|---|
| `ExpiredToken` or `InvalidClientTokenId` | The lab session ended. Restart it and redo [step 3](#3-set-your-aws-credentials). |
| `Cannot connect to the Docker daemon` | Start Docker Desktop and wait until `docker info` works, then redo the `export DOCKER_HOST=...` line from [step 4](#4-start-docker) in the same terminal. |
| `curl` times out | The task may still be starting, or it restarted with a new IP. Wait a minute and redo [step 6](#6-get-the-servers-public-ip). |
| `400` with `invalid character '\n' in string literal` | A line break got pasted inside a JSON string. Copy the request again as one block from this README. |
| Task keeps stopping | Check the logs: `aws logs tail /ecs/CS6650L2 --region us-west-2 --since 15m` |
| `RepositoryAlreadyExistsException` or `ResourceAlreadyExists` | A previous deploy in this lab account wasn't destroyed. Run `terraform destroy` from the same folder, or delete the leftover resources in the AWS console. |

The Docker image is built for `linux/amd64`, the architecture Fargate uses by default, so it deploys correctly from both Intel and Apple Silicon Macs.
