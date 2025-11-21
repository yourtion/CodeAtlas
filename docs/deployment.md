# 部署指南

> 将 CodeAtlas 部署到生产环境

## 概述

CodeAtlas 支持多种部署方式：
- Docker Compose（单机部署）
- Kubernetes（集群部署）
- 云服务（AWS、GCP、Azure）

## Docker Compose 部署

### 生产环境配置

创建 `docker-compose.prod.yml`：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:17-bookworm
    restart: always
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./deployments/migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  api:
    build:
      context: .
      dockerfile: deployments/Dockerfile.api
    restart: always
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - API_PORT=8080
      - LOG_LEVEL=warn
      - LOG_FORMAT=json
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
```

### 启动服务

```bash
# 1. 配置环境变量
cp .env.example .env.production
vim .env.production

# 2. 启动服务
docker-compose -f docker-compose.prod.yml up -d

# 3. 检查状态
docker-compose -f docker-compose.prod.yml ps

# 4. 查看日志
docker-compose -f docker-compose.prod.yml logs -f
```

### 数据持久化

```bash
# 备份数据库
docker-compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U codeatlas codeatlas > backup.sql

# 恢复数据库
docker-compose -f docker-compose.prod.yml exec -T postgres \
  psql -U codeatlas codeatlas < backup.sql
```

## Kubernetes 部署

### 前置要求

- Kubernetes 1.20+
- kubectl 配置完成
- Helm 3.0+（可选）

### 创建命名空间

```bash
kubectl create namespace codeatlas
```

### 配置 Secret

```bash
# 创建数据库密码
kubectl create secret generic codeatlas-db \
  --from-literal=username=codeatlas \
  --from-literal=password=your-secure-password \
  -n codeatlas

# 创建 API Key
kubectl create secret generic codeatlas-api \
  --from-literal=api-key=your-api-key \
  --from-literal=openai-key=your-openai-key \
  -n codeatlas
```

### PostgreSQL 部署

创建 `k8s/postgres.yaml`：

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: codeatlas
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: codeatlas
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:17-bookworm
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: codeatlas-db
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: codeatlas-db
              key: password
        - name: POSTGRES_DB
          value: codeatlas
        volumeMounts:
        - name: postgres-storage
          mountPath: /var/lib/postgresql/data
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
          limits:
            memory: "4Gi"
            cpu: "2000m"
      volumes:
      - name: postgres-storage
        persistentVolumeClaim:
          claimName: postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: codeatlas
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
  clusterIP: None
```

### API 部署

创建 `k8s/api.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: codeatlas-api
  namespace: codeatlas
spec:
  replicas: 3
  selector:
    matchLabels:
      app: codeatlas-api
  template:
    metadata:
      labels:
        app: codeatlas-api
    spec:
      containers:
      - name: api
        image: codeatlas/api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DB_HOST
          value: postgres
        - name: DB_PORT
          value: "5432"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: codeatlas-db
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: codeatlas-db
              key: password
        - name: DB_NAME
          value: codeatlas
        - name: API_PORT
          value: "8080"
        - name: LOG_LEVEL
          value: warn
        - name: LOG_FORMAT
          value: json
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: codeatlas-api
  namespace: codeatlas
spec:
  selector:
    app: codeatlas-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### 部署到 Kubernetes

```bash
# 部署 PostgreSQL
kubectl apply -f k8s/postgres.yaml

# 等待 PostgreSQL 就绪
kubectl wait --for=condition=ready pod -l app=postgres -n codeatlas --timeout=300s

# 部署 API
kubectl apply -f k8s/api.yaml

# 检查状态
kubectl get pods -n codeatlas
kubectl get svc -n codeatlas

# 查看日志
kubectl logs -f deployment/codeatlas-api -n codeatlas
```

### Ingress 配置

创建 `k8s/ingress.yaml`：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: codeatlas-ingress
  namespace: codeatlas
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.codeatlas.example.com
    secretName: codeatlas-tls
  rules:
  - host: api.codeatlas.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: codeatlas-api
            port:
              number: 80
```

```bash
kubectl apply -f k8s/ingress.yaml
```

## 云服务部署

### AWS 部署

#### 使用 ECS

```bash
# 1. 创建 ECR 仓库
aws ecr create-repository --repository-name codeatlas/api

# 2. 构建并推送镜像
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin <account-id>.dkr.ecr.us-east-1.amazonaws.com

docker build -t codeatlas/api -f deployments/Dockerfile.api .
docker tag codeatlas/api:latest <account-id>.dkr.ecr.us-east-1.amazonaws.com/codeatlas/api:latest
docker push <account-id>.dkr.ecr.us-east-1.amazonaws.com/codeatlas/api:latest

# 3. 创建 RDS 数据库
aws rds create-db-instance \
  --db-instance-identifier codeatlas-db \
  --db-instance-class db.t3.medium \
  --engine postgres \
  --engine-version 17 \
  --master-username codeatlas \
  --master-user-password <password> \
  --allocated-storage 100

# 4. 创建 ECS 集群和服务
# 使用 AWS Console 或 CloudFormation
```

#### 使用 EKS

```bash
# 1. 创建 EKS 集群
eksctl create cluster \
  --name codeatlas \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3

# 2. 配置 kubectl
aws eks update-kubeconfig --name codeatlas --region us-east-1

# 3. 部署应用
kubectl apply -f k8s/
```

### GCP 部署

#### 使用 Cloud Run

```bash
# 1. 构建镜像
gcloud builds submit --tag gcr.io/PROJECT_ID/codeatlas-api

# 2. 部署到 Cloud Run
gcloud run deploy codeatlas-api \
  --image gcr.io/PROJECT_ID/codeatlas-api \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars DB_HOST=<cloud-sql-ip>

# 3. 创建 Cloud SQL 实例
gcloud sql instances create codeatlas-db \
  --database-version=POSTGRES_17 \
  --tier=db-f1-micro \
  --region=us-central1
```

### Azure 部署

#### 使用 Container Instances

```bash
# 1. 创建资源组
az group create --name codeatlas-rg --location eastus

# 2. 创建 PostgreSQL
az postgres flexible-server create \
  --resource-group codeatlas-rg \
  --name codeatlas-db \
  --location eastus \
  --admin-user codeatlas \
  --admin-password <password> \
  --sku-name Standard_B1ms

# 3. 部署容器
az container create \
  --resource-group codeatlas-rg \
  --name codeatlas-api \
  --image codeatlas/api:latest \
  --dns-name-label codeatlas-api \
  --ports 8080 \
  --environment-variables \
    DB_HOST=codeatlas-db.postgres.database.azure.com \
    DB_USER=codeatlas \
    DB_PASSWORD=<password>
```

## 监控和日志

### Prometheus 监控

创建 `k8s/monitoring.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: codeatlas
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
    - job_name: 'codeatlas-api'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - codeatlas
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: codeatlas-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prometheus
  namespace: codeatlas
spec:
  replicas: 1
  selector:
    matchLabels:
      app: prometheus
  template:
    metadata:
      labels:
        app: prometheus
    spec:
      containers:
      - name: prometheus
        image: prom/prometheus:latest
        ports:
        - containerPort: 9090
        volumeMounts:
        - name: config
          mountPath: /etc/prometheus
      volumes:
      - name: config
        configMap:
          name: prometheus-config
```

### Grafana 仪表板

```bash
# 部署 Grafana
kubectl apply -f k8s/grafana.yaml

# 访问 Grafana
kubectl port-forward svc/grafana 3000:3000 -n codeatlas
```

### 日志聚合

#### 使用 ELK Stack

```yaml
# k8s/elasticsearch.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: elasticsearch
  namespace: codeatlas
spec:
  serviceName: elasticsearch
  replicas: 1
  selector:
    matchLabels:
      app: elasticsearch
  template:
    metadata:
      labels:
        app: elasticsearch
    spec:
      containers:
      - name: elasticsearch
        image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
        ports:
        - containerPort: 9200
        env:
        - name: discovery.type
          value: single-node
```

## 备份和恢复

### 自动备份

创建备份脚本 `scripts/backup.sh`：

```bash
#!/bin/bash
# 数据库备份脚本

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/codeatlas_$DATE.sql"

# 创建备份
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME > $BACKUP_FILE

# 压缩备份
gzip $BACKUP_FILE

# 上传到 S3
aws s3 cp $BACKUP_FILE.gz s3://codeatlas-backups/

# 清理旧备份（保留 30 天）
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete
```

### 定时备份

```yaml
# k8s/backup-cronjob.yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: database-backup
  namespace: codeatlas
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:17
            command:
            - /bin/sh
            - -c
            - |
              pg_dump -h postgres -U codeatlas codeatlas | \
              gzip > /backups/backup_$(date +%Y%m%d_%H%M%S).sql.gz
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: codeatlas-db
                  key: password
            volumeMounts:
            - name: backup-storage
              mountPath: /backups
          restartPolicy: OnFailure
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
```

### 恢复数据

```bash
# 从备份恢复
gunzip -c backup.sql.gz | \
  psql -h $DB_HOST -U $DB_USER -d $DB_NAME

# Kubernetes 环境
kubectl exec -it postgres-0 -n codeatlas -- \
  psql -U codeatlas -d codeatlas < backup.sql
```

## 安全最佳实践

### 1. 网络安全

```yaml
# k8s/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: codeatlas-network-policy
  namespace: codeatlas
spec:
  podSelector:
    matchLabels:
      app: codeatlas-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx-ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
```

### 2. 密钥管理

```bash
# 使用 Sealed Secrets
kubectl apply -f https://github.com/bitnami-labs/sealed-secrets/releases/download/v0.18.0/controller.yaml

# 创建加密的 Secret
kubeseal --format yaml < secret.yaml > sealed-secret.yaml
kubectl apply -f sealed-secret.yaml
```

### 3. RBAC 配置

```yaml
# k8s/rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: codeatlas-api
  namespace: codeatlas
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: codeatlas-api-role
  namespace: codeatlas
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: codeatlas-api-rolebinding
  namespace: codeatlas
subjects:
- kind: ServiceAccount
  name: codeatlas-api
roleRef:
  kind: Role
  name: codeatlas-api-role
  apiGroup: rbac.authorization.k8s.io
```

## 性能优化

### 1. 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_symbols_name ON symbols(name);
CREATE INDEX idx_symbols_repo_id ON symbols(repo_id);
CREATE INDEX idx_vectors_embedding ON vectors USING ivfflat (embedding vector_cosine_ops);

-- 配置连接池
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET shared_buffers = '2GB';
ALTER SYSTEM SET effective_cache_size = '6GB';
```

### 2. API 缓存

```yaml
# 使用 Redis 缓存
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  namespace: codeatlas
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7-alpine
        ports:
        - containerPort: 6379
```

### 3. 负载均衡

```yaml
# 配置 HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: codeatlas-api-hpa
  namespace: codeatlas
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: codeatlas-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## 故障排除

### 检查服务状态

```bash
# Docker Compose
docker-compose ps
docker-compose logs api

# Kubernetes
kubectl get pods -n codeatlas
kubectl describe pod <pod-name> -n codeatlas
kubectl logs <pod-name> -n codeatlas
```

### 常见问题

**数据库连接失败**：
```bash
# 检查网络
kubectl exec -it <api-pod> -n codeatlas -- ping postgres

# 检查密码
kubectl get secret codeatlas-db -n codeatlas -o yaml
```

**内存不足**：
```bash
# 增加资源限制
kubectl set resources deployment codeatlas-api \
  --limits=memory=2Gi,cpu=2000m \
  -n codeatlas
```

## 下一步

- 查看 [配置指南](configuration.md) 了解详细配置
- 查看 [监控指南](monitoring.md) 设置监控
- 查看 [故障排除](troubleshooting.md) 解决问题
