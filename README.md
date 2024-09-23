# Kubernetes TokenReview Mock Server

A mock server implementing the Kubernetes TokenReview endpoint, designed for testing HashiCorp Vault's Kubernetes auth engine.

## Quick Start

### Running with Docker

Run the Docker container:
```bash
$ docker pull hedisam/kubeservermock:latest
$ docker run -it --rm -p 6443:6443 hedisam/kubeservermock:latest
```
The mock server will be available at `http://localhost:6443`.

## Usage

This mock server provides the following key endpoints:

1. TokenReview Endpoint (Simulates Kubernetes API):
   - `POST /apis/authentication.k8s.io/v1/tokenreviews`
   - Used by Vault to validate service account tokens
2. Service Account Registration (For testing):
   - `POST /api/v1/testing/serviceaccounts`
   - Register a test service account and get a JWT token
3. Health Status (For testing):
   - `GET /api/v1/testing/health`
   - Query this endpoint to make sure the container is up and healthy
4. Reset State:
   - `DELETE /api/v1/testing/reset`
   - Clear a specific or all registered service accounts

### Example Workflow

1. Register a service account:
```bash
$ curl -X POST http://localhost:6443/api/v1/testing/serviceaccounts -d '{"name":"my-service","namespace":"default","uid":"12345"}'
```
This returns a valid JWT token that a testing HC Vault instance can accept. In an environment with a real k8s server and pod, this JWT token will be the same as the one mounted at `/var/run/secrets/kubernetes.io/serviceaccount/token`
```json
{
  "jwt": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L25hbWVzcGFjZSI6ImRlZmF1bHQiLCJrdWJlcm5ldGVzLmlvL3NlcnZpY2VhY2NvdW50L3NlcnZpY2UtYWNjb3VudC5uYW1lIjoibXktc2VydmljZSIsImt1YmVybmV0ZXMuaW8vc2VydmljZWFjY291bnQvc2VydmljZS1hY2NvdW50LnVpZCI6IjEyMzQ1In0.EF0ZZ94lO4w8i87Eh9qPFR24VuwH82PTZD2kRh6wa4afQFILZpdo7jXDB6x1uv98DETbGqENFK9nTYVyvWZX9Y76C4wxhg8uPr-eG2oviD7LtRzyoo4-21wAkh_crytj0JXrtEbcjta4ar3jJMzAaJW6ofsfrVZ4cpzDjOAvO36qLjvfN6wyB29lWG9tkqmlUar1tgvSBU97pCon2b7obipW-TGV1UxuUObV4Sc_kcnk0tm0VubXsMOR1oDKVSWCy5HDFFa89Dm3-J--805M0kETwGIxlITcrtRUgfRHKn6fe9yiXWjmGBl2kBlWOc6QeGRCDHQ0VLdX17a2Si5WPA",
  "success": true
}
```

2. Use the JWT token with Vault's Kubernetes auth method for testing.
3. The testing HC Vault instance will use the JWT token to validate & login with the mock kube server  
4. Reset the mock server state between tests if necessary:
```bash
$ curl -X DELETE http://localhost:6443/api/v1/testing/reset
```