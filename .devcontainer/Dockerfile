FROM golang:1.25-bookworm

# Install system dependencies
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    git \
    curl \
    wget \
    ca-certificates \
    postgresql-client \
    build-essential \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go development tools.
# 各工具独立安装并容错（|| true）：这些是开发辅助工具（lint/LSP/debug），
# 单个安装失败不应阻塞容器构建——CI 只需代码能编译测试即可。
# golangci-lint 用官方安装脚本（避免 go install @latest 在新 Go 版本下的依赖冲突）。
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
        | sh -s -- -b $(go env GOPATH)/bin \
    || echo "warning: golangci-lint install failed (non-fatal)"
RUN go install golang.org/x/tools/gopls@latest || echo "warning: gopls install failed (non-fatal)"
RUN go install github.com/go-delve/delve/cmd/dlv@latest || echo "warning: dlv install failed (non-fatal)"
RUN go install honnef.co/go/tools/cmd/staticcheck@latest || echo "warning: staticcheck install failed (non-fatal)"

# Create non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && apt-get update \
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

USER $USERNAME
WORKDIR /workspace
