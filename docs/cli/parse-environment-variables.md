# Parse Command Environment Variables

This document describes all environment variables that affect the `codeatlas parse` command.

## LLM Enhancement Variables

These variables are only used when the `--semantic` flag is enabled.

### CODEATLAS_LLM_API_KEY

**Description**: API key for LLM service (OpenAI or compatible)

**Required**: Yes, when using `--semantic` flag

**Default**: None

**Example**:
```bash
export CODEATLAS_LLM_API_KEY=sk-proj-abc123xyz789...
```

**Usage**:
```bash
export CODEATLAS_LLM_API_KEY=sk-your-key
codeatlas parse --path ./project --semantic
```

**Notes**:
- Keep this secret and never commit to version control
- Add to `.bashrc` or `.zshrc` for persistent configuration
- Can also be set in `.env` file (if supported)

---

### CODEATLAS_LLM_API_URL

**Description**: Custom LLM API endpoint URL

**Required**: No

**Default**: `https://api.openai.com/v1`

**Example**:
```bash
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1
```

**Usage**:
```bash
# Use Azure OpenAI
export CODEATLAS_LLM_API_URL=https://your-resource.openai.azure.com/openai/deployments/your-deployment

# Use local LLM server
export CODEATLAS_LLM_API_URL=http://localhost:8000/v1

# Use custom endpoint
export CODEATLAS_LLM_API_URL=https://your-custom-llm.com/v1
```

**Notes**:
- Must be compatible with OpenAI API format
- Include `/v1` suffix if required by your endpoint
- Useful for self-hosted LLMs or alternative providers

---

### CODEATLAS_LLM_MODEL

**Description**: LLM model name to use for semantic enhancement

**Required**: No

**Default**: `gpt-3.5-turbo`

**Example**:
```bash
export CODEATLAS_LLM_MODEL=gpt-4
```

**Usage**:
```bash
# Use GPT-4
export CODEATLAS_LLM_MODEL=gpt-4

# Use GPT-3.5 Turbo (faster, cheaper)
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo

# Use custom model
export CODEATLAS_LLM_MODEL=my-custom-model
```

**Supported Models**:
- `gpt-4` - Most capable, slower, more expensive
- `gpt-4-turbo` - Faster GPT-4 variant
- `gpt-3.5-turbo` - Fast, cost-effective (default)
- Custom models (if using compatible API)

**Notes**:
- Different models have different costs and rate limits
- Choose based on your accuracy vs. speed requirements
- GPT-3.5-turbo is recommended for most use cases

---

## Configuration Examples

### Basic Setup (OpenAI)

```bash
# Add to ~/.bashrc or ~/.zshrc
export CODEATLAS_LLM_API_KEY=sk-your-openai-key

# Use in parse command
codeatlas parse --path ./project --semantic
```

### Azure OpenAI Setup

```bash
export CODEATLAS_LLM_API_KEY=your-azure-key
export CODEATLAS_LLM_API_URL=https://your-resource.openai.azure.com/openai/deployments/your-deployment
export CODEATLAS_LLM_MODEL=gpt-35-turbo

codeatlas parse --path ./project --semantic
```

### Local LLM Setup (e.g., Ollama)

```bash
export CODEATLAS_LLM_API_URL=http://localhost:11434/v1
export CODEATLAS_LLM_MODEL=llama2

# No API key needed for local
codeatlas parse --path ./project --semantic
```

### Custom Provider Setup

```bash
export CODEATLAS_LLM_API_KEY=your-provider-key
export CODEATLAS_LLM_API_URL=https://api.your-provider.com/v1
export CODEATLAS_LLM_MODEL=provider-model-name

codeatlas parse --path ./project --semantic
```

---

## Environment File (.env)

You can create a `.env` file in your project root:

```bash
# .env
CODEATLAS_LLM_API_KEY=sk-your-key
CODEATLAS_LLM_API_URL=https://api.openai.com/v1
CODEATLAS_LLM_MODEL=gpt-3.5-turbo
```

Load it before running:
```bash
# Load environment variables
export $(cat .env | xargs)

# Run parse command
codeatlas parse --path ./project --semantic
```

**Security Note**: Add `.env` to `.gitignore` to prevent committing secrets:
```bash
echo ".env" >> .gitignore
```

---

## Verification

### Check if variables are set

```bash
# Check all LLM variables
echo "API Key: ${CODEATLAS_LLM_API_KEY:0:10}..." # Shows first 10 chars
echo "API URL: $CODEATLAS_LLM_API_URL"
echo "Model: $CODEATLAS_LLM_MODEL"
```

### Test API connection

```bash
# Test OpenAI API
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY"

# Test custom endpoint
curl $CODEATLAS_LLM_API_URL/models \
  -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY"
```

---

## Troubleshooting

### API Key Not Found

**Symptom**:
```
[ERROR] LLM API error: invalid API key
```

**Solution**:
```bash
# Check if set
echo $CODEATLAS_LLM_API_KEY

# If empty, set it
export CODEATLAS_LLM_API_KEY=sk-your-key

# Verify
echo ${CODEATLAS_LLM_API_KEY:0:10}...
```

### Wrong API Endpoint

**Symptom**:
```
[ERROR] LLM API error: connection refused
[ERROR] LLM API error: 404 not found
```

**Solution**:
```bash
# Check current value
echo $CODEATLAS_LLM_API_URL

# Set correct endpoint
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1

# Test connection
curl -I $CODEATLAS_LLM_API_URL/models
```

### Model Not Available

**Symptom**:
```
[ERROR] LLM API error: model not found
```

**Solution**:
```bash
# Check current model
echo $CODEATLAS_LLM_MODEL

# List available models
curl $CODEATLAS_LLM_API_URL/models \
  -H "Authorization: Bearer $CODEATLAS_LLM_API_KEY" | jq '.data[].id'

# Set available model
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo
```

---

## Best Practices

### 1. Use Shell Configuration Files

Add to `~/.bashrc` or `~/.zshrc` for persistent configuration:

```bash
# Add to ~/.bashrc
cat >> ~/.bashrc << 'EOF'
# CodeAtlas LLM Configuration
export CODEATLAS_LLM_API_KEY=sk-your-key
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo
EOF

# Reload
source ~/.bashrc
```

### 2. Use Different Profiles

Create profile scripts for different environments:

```bash
# ~/.codeatlas-openai
export CODEATLAS_LLM_API_KEY=sk-openai-key
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo

# ~/.codeatlas-azure
export CODEATLAS_LLM_API_KEY=azure-key
export CODEATLAS_LLM_API_URL=https://your-resource.openai.azure.com/...
export CODEATLAS_LLM_MODEL=gpt-35-turbo

# Load profile
source ~/.codeatlas-openai
codeatlas parse --path ./project --semantic
```

### 3. Secure API Keys

```bash
# Set restrictive permissions on config files
chmod 600 ~/.codeatlas-openai

# Never commit API keys
echo ".env" >> .gitignore
echo "*.key" >> .gitignore

# Use environment-specific keys
# - Development: Limited rate, test key
# - Production: Full access, production key
```

### 4. Cost Management

```bash
# Use cheaper model for development
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo

# Use better model for production
export CODEATLAS_LLM_MODEL=gpt-4

# Monitor usage
# Check your API provider's dashboard regularly
```

---

## Future Environment Variables

These variables may be added in future versions:

- `CODEATLAS_LLM_TIMEOUT` - API request timeout
- `CODEATLAS_LLM_RATE_LIMIT` - Custom rate limit
- `CODEATLAS_LLM_MAX_TOKENS` - Maximum tokens per request
- `CODEATLAS_CACHE_DIR` - Cache directory for parsed results
- `CODEATLAS_LOG_LEVEL` - Logging verbosity (debug, info, warn, error)

---

## Related Documentation

- [CLI Parse Command Documentation](./cli-parse-command.md)
- [Troubleshooting Guide](./parse-troubleshooting.md)
- [Quick Reference](./parse-command-quick-reference.md)

---

## Summary Table

| Variable | Required | Default | Purpose |
|----------|----------|---------|---------|
| `CODEATLAS_LLM_API_KEY` | Yes (with --semantic) | None | API authentication |
| `CODEATLAS_LLM_API_URL` | No | `https://api.openai.com/v1` | API endpoint |
| `CODEATLAS_LLM_MODEL` | No | `gpt-3.5-turbo` | Model selection |

---

## Quick Start

```bash
# Minimal setup
export CODEATLAS_LLM_API_KEY=sk-your-key
codeatlas parse --path ./project --semantic

# Full setup
export CODEATLAS_LLM_API_KEY=sk-your-key
export CODEATLAS_LLM_API_URL=https://api.openai.com/v1
export CODEATLAS_LLM_MODEL=gpt-3.5-turbo
codeatlas parse --path ./project --semantic --output result.json
```
