# 使用 PostgreSQL 17 官方 Debian 基础镜像
FROM postgres:17-bookworm

# 安装 Pigsty APT 仓库并安装 pgvector + age
RUN apt-get update && apt-get install -y curl gnupg lsb-release \
    && curl -fsSL https://repo.pigsty.cc/key | gpg --dearmor -o /etc/apt/keyrings/pigsty.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/pigsty.gpg] https://repo.pigsty.cc/apt/infra generic main" \
            > /etc/apt/sources.list.d/pigsty.list \
    && echo "deb [signed-by=/etc/apt/keyrings/pigsty.gpg] https://repo.pigsty.cc/apt/pgsql/$(lsb_release -cs) $(lsb_release -cs) main" \
            >> /etc/apt/sources.list.d/pigsty.list \
    && apt-get update \
    && apt-get install -y postgresql-17-pgvector postgresql-17-age \
    && rm -rf /var/lib/apt/lists/*

# 默认挂载 PG 数据目录
VOLUME ["/var/lib/postgresql/data"]

# 默认启动 PostgreSQL
CMD ["postgres"]

