# Jan API Gateway Helm Chart

A comprehensive Helm chart for deploying the Jan API Gateway with production-ready features including PostgreSQL, Valkey cache, OAuth2, SMTP, and advanced autoscaling.

## Features

- **Production-ready Kubernetes resources**: Deployment, Service, Ingress, ConfigMaps, Secrets
- **Database Integration**: CloudNativePG PostgreSQL operator or external PostgreSQL  
- **Caching**: Valkey (Redis-compatible) cluster or external Redis
- **Authentication**: OAuth2 Google integration with secure secret management
- **Email Services**: SMTP support (SendGrid, etc.) with encrypted credentials
- **Autoscaling**: HPA, VPA, and KEDA support for different scaling strategies
- **Security**: Pod Security Contexts, Service Accounts, Pod Disruption Budgets
- **Observability**: Ready for Prometheus metrics collection

## Prerequisites

- Kubernetes 1.20+
- Helm 3.8+
- Optional: CloudNativePG operator for PostgreSQL
- Optional: KEDA operator for event-driven autoscaling
- Optional: VPA for vertical pod autoscaling

## Installation

### Quick Start

```bash
# Add required Helm repositories
helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo add oci://registry-1.docker.io/bitnamicharts

# Update repositories
helm repo update

# Install with dependencies
helm dependency update ./jan-api-gateway
helm install jan-api-gateway ./jan-api-gateway
```

### Production Installation

```bash
# Create your production values file
cp values.yaml values-prod.yaml
# Edit values-prod.yaml with your configuration

# Install with custom values
helm install jan-api-gateway ./jan-api-gateway -f values-prod.yaml
```

## Configuration

### Application Settings

```yaml
janApiGateway:
  image:
    repository: your-registry/jan-api-gateway
    tag: "v1.0.0"
  
  baseUrl: "https://api.yourcompany.com"
  inferenceBaseUrl: "http://jan-inference-model:8000"
  
  # OAuth2 Google Configuration
  oauth2:
    enabled: true
    google:
      clientId: "your-google-client-id"
      clientSecret: "your-google-client-secret"
      redirectUrl: "https://api.yourcompany.com/api/v1/auth/google/callback"
  
  # SMTP Configuration (SendGrid example)
  smtp:
    enabled: true
    host: "smtp.sendgrid.net"
    port: "587"
    username: "apikey"
    password: "your-sendgrid-api-key"
    fromEmail: "noreply@yourcompany.com"
  
  # Application Secrets
  secrets:
    jwtSecret: "your-32-char-jwt-secret-key-here"
    apiKeySecret: "your-api-key-secret"
    serperApiKey: "your-serper-api-key"
    adminEmail: "admin@yourcompany.com"
```

### Database Configuration

#### CloudNativePG (Recommended)

```yaml
cloudnative-pg:
  enabled: true
  cluster:
    instances: 3  # For high availability
    postgresql:
      database: "jan_api_gateway"
      owner: "jan_user"
    storage:
      size: "100Gi"
      storageClass: "fast-ssd"
```

#### External PostgreSQL

```yaml
cloudnative-pg:
  enabled: false

externalPostgresql:
  host: "your-postgres-host"
  port: 5432
  database: "jan_api_gateway"
  username: "jan_user"
  password: "your-password"
```

### Caching Configuration

#### Valkey Cluster (Recommended)

```yaml
valkey:
  enabled: true
  auth:
    enabled: true
    password: "your-strong-redis-password"
```

#### External Redis/Valkey

```yaml
valkey:
  enabled: false

externalValkey:
  host: "your-redis-host"
  port: 6379
  password: "your-redis-password"
```

### Autoscaling Configuration

#### Horizontal Pod Autoscaler (HPA)

```yaml
janApiGateway:
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 50
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
```

#### KEDA (Event-driven Autoscaling)

```yaml
janApiGateway:
  keda:
    enabled: true
    minReplicas: 2
    maxReplicas: 100
    triggers:
      - type: prometheus
        metadata:
          serverAddress: http://prometheus:9090
          metricName: http_requests_per_second
          threshold: '100'
          query: sum(rate(http_requests_total[1m]))
```

#### Vertical Pod Autoscaler (VPA)

```yaml
janApiGateway:
  vpa:
    enabled: true
    updateMode: "Auto"
    maxAllowed:
      cpu: 2000m
      memory: 4Gi
```

### Ingress Configuration

```yaml
janApiGateway:
  ingress:
    enabled: true
    className: "nginx"
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
      nginx.ingress.kubernetes.io/rate-limit: "100"
      nginx.ingress.kubernetes.io/ssl-redirect: "true"
    hosts:
      - host: api.yourcompany.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: api-tls-secret
        hosts:
          - api.yourcompany.com
```

## Environment Variables

The chart automatically configures these environment variables for your application:

### Database Connection
- `DB_HOST` - PostgreSQL host
- `DB_PORT` - PostgreSQL port (5432)
- `DB_NAME` - Database name
- `DB_USERNAME` - Database username (from secret)
- `DB_PASSWORD` - Database password (from secret)
- `DATABASE_URL` - Complete PostgreSQL connection string

### Cache Connection  
- `REDIS_HOST` - Redis/Valkey host
- `REDIS_PORT` - Redis/Valkey port (6379)
- `REDIS_PASSWORD` - Redis/Valkey password (from secret)
- `REDIS_URL` - Complete Redis connection string

### OAuth2 Configuration
- `GOOGLE_CLIENT_ID` - Google OAuth client ID (from secret)
- `GOOGLE_CLIENT_SECRET` - Google OAuth client secret (from secret)
- `GOOGLE_OAUTH_REDIRECT_URL` - OAuth callback URL

### SMTP Configuration
- `SMTP_HOST` - SMTP server host
- `SMTP_PORT` - SMTP server port
- `SMTP_USERNAME` - SMTP username (from secret)
- `SMTP_PASSWORD` - SMTP password/API key (from secret)
- `SMTP_FROM_EMAIL` - From email address

### Application Secrets
- `JWT_SECRET` - JWT signing secret (from secret)
- `APIKEY_SECRET` - API key secret (from secret)
- `SERPER_API_KEY` - Serper.dev API key (from secret)

### Other Configuration
- `ADMIN_EMAIL` - Admin email address
- `GATEWAY_BASE_URL` - Gateway base URL
- `INFERENCE_BASE_URL` - Inference service URL
- `PORT` - Application port (8080)

## Security Best Practices

### Using Existing Secrets

For production, create secrets manually instead of using plain text values:

```bash
# Create OAuth2 secret
kubectl create secret generic oauth2-credentials \
  --from-literal=google-client-id="your-client-id" \
  --from-literal=google-client-secret="your-client-secret"

# Create SMTP secret  
kubectl create secret generic smtp-credentials \
  --from-literal=smtp-username="apikey" \
  --from-literal=smtp-password="your-sendgrid-key"

# Create application secrets
kubectl create secret generic app-secrets \
  --from-literal=jwt-secret="your-jwt-secret" \
  --from-literal=apikey-secret="your-api-secret" \
  --from-literal=serper-api-key="your-serper-key"
```

Then reference them in your values:

```yaml
janApiGateway:
  oauth2:
    existingSecret: "oauth2-credentials"
  smtp:
    existingSecret: "smtp-credentials"
  secrets:
    existingSecret: "app-secrets"
```

## Dependencies

This chart includes the following dependencies:

- **CloudNativePG v1.27.0** - Modern PostgreSQL operator for Kubernetes
- **Valkey Cluster v3.0.24** - High-performance Redis-compatible cache

## Monitoring & Observability

The application is ready for monitoring with:

- Prometheus metrics endpoint: `/metrics`
- Health check endpoints: `/health`, `/ready`
- Structured logging with configurable levels
- Request tracing support

## Examples

### Development Environment

```yaml
# values-dev.yaml
janApiGateway:
  image:
    tag: "develop"
  
  # Disable auth for development
  oauth2:
    enabled: false
  
  # Use simple logging
  extraEnv:
    - name: LOG_LEVEL
      value: "debug"

# Smaller resources for development
cloudnative-pg:
  cluster:
    instances: 1
    storage:
      size: "10Gi"

valkey:
  auth:
    enabled: false
```

### Production Environment

```yaml
# values-prod.yaml
janApiGateway:
  replicaCount: 5
  
  image:
    tag: "v1.2.3"
  
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2000m
      memory: 2Gi
  
  # Production autoscaling
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 50
    targetCPUUtilizationPercentage: 70
  
  # Production ingress
  ingress:
    enabled: true
    className: "nginx"
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
    hosts:
      - host: api.yourcompany.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: api-tls
        hosts:
          - api.yourcompany.com

# High availability database
cloudnative-pg:
  cluster:
    instances: 3
    storage:
      size: "500Gi"
      storageClass: "fast-ssd"

# High availability cache
valkey:
  auth:
    enabled: true
    password: "strong-production-password"
```

## Troubleshooting

### Check Application Status

```bash
# Check pods
kubectl get pods -l app.kubernetes.io/name=jan-api-gateway

# Check logs
kubectl logs -l app.kubernetes.io/name=jan-api-gateway -f

# Check service
kubectl get svc jan-api-gateway
```

### Database Troubleshooting

```bash
# Check PostgreSQL cluster status
kubectl get cluster

# Check PostgreSQL pods
kubectl get pods -l postgres-operator.crunchydata.com/cluster=jan-api-gateway

# Check database credentials
kubectl get secret jan-api-gateway-postgresql-app -o yaml
```

### Cache Troubleshooting

```bash
# Check Valkey cluster status
kubectl get pods -l app.kubernetes.io/name=valkey

# Test Redis connection
kubectl exec -it <valkey-pod> -- redis-cli ping
```

### Environment Variables Check

```bash
# Check all environment variables in the pod
kubectl exec -it <pod-name> -- env | sort

# Check specific database vars
kubectl exec -it <pod-name> -- env | grep DB_

# Check Redis vars
kubectl exec -it <pod-name> -- env | grep REDIS_
```

## Upgrading

### Standard Upgrade

```bash
# Update dependencies
helm dependency update

# Upgrade release
helm upgrade jan-api-gateway ./jan-api-gateway -f values-prod.yaml
```

### Rolling Back

```bash
# Check release history
helm history jan-api-gateway

# Rollback to previous version
helm rollback jan-api-gateway 1
```

## Contributing

1. Fork the repository
2. Make your changes to templates or values
3. Test with `helm template` and `helm lint`
4. Test installation in development cluster
5. Submit pull request

## Support

For issues and questions:

- Check the troubleshooting section above
- Review Kubernetes events: `kubectl get events`
- Check application logs: `kubectl logs -f deployment/jan-api-gateway`

## License

This Helm chart is licensed under the Apache 2.0 License.