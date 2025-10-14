# Parse Command Documentation Index

Complete documentation for the `codeatlas parse` command.

## Quick Links

- **[Complete Documentation](./cli-parse-command.md)** - Start here for comprehensive guide
- **[Quick Reference](./parse-command-quick-reference.md)** - Common commands at a glance
- **[Troubleshooting](./parse-troubleshooting.md)** - Fix common issues
- **[Environment Variables](./parse-environment-variables.md)** - Configuration options
- **[Example Output](./examples/parse-output-example.json)** - See what the output looks like

## Documentation Overview

### 1. Complete Documentation (cli-parse-command.md)
**Purpose**: Comprehensive reference guide  
**Contents**:
- Installation instructions
- All command-line flags with examples
- Output format specification
- Usage examples for common scenarios
- Performance tips
- Supported languages and features

**When to use**: First-time users, detailed reference, understanding all features

### 2. Quick Reference (parse-command-quick-reference.md)
**Purpose**: Fast lookup for common commands  
**Contents**:
- Common command patterns
- Language-specific parsing
- Performance optimization commands
- Debugging commands
- Output processing with jq
- Quick troubleshooting fixes

**When to use**: Experienced users, quick command lookup, copy-paste examples

### 3. Troubleshooting Guide (parse-troubleshooting.md)
**Purpose**: Diagnose and fix problems  
**Contents**:
- No files found issues
- Syntax error handling
- Permission problems
- Performance issues
- LLM API errors
- Output problems
- Memory issues
- Ignore pattern problems

**When to use**: Encountering errors, unexpected behavior, performance problems

### 4. Environment Variables (parse-environment-variables.md)
**Purpose**: Configure LLM enhancement  
**Contents**:
- CODEATLAS_LLM_API_KEY
- CODEATLAS_LLM_API_URL
- CODEATLAS_LLM_MODEL
- Configuration examples for different providers
- Security best practices

**When to use**: Setting up semantic enhancement, using custom LLM providers

### 5. Example Output (examples/parse-output-example.json)
**Purpose**: Understand output format  
**Contents**:
- Complete JSON output example
- Multiple languages (Go, JavaScript, Python)
- Files, symbols, relationships, metadata
- Real-world structure

**When to use**: Understanding output schema, building integrations, debugging output

## Learning Path

### Beginner
1. Read [Complete Documentation](./cli-parse-command.md) - Overview and basic usage
2. Try basic commands from [Quick Reference](./parse-command-quick-reference.md)
3. Review [Example Output](./examples/parse-output-example.json) to understand results

### Intermediate
1. Explore advanced flags in [Complete Documentation](./cli-parse-command.md)
2. Set up [Environment Variables](./parse-environment-variables.md) for LLM enhancement
3. Optimize performance using tips from [Quick Reference](./parse-command-quick-reference.md)

### Advanced
1. Process output with jq (see [Quick Reference](./parse-command-quick-reference.md))
2. Integrate with CI/CD (see [Complete Documentation](./cli-parse-command.md))
3. Customize ignore patterns and filters

### Troubleshooting
1. Check [Troubleshooting Guide](./parse-troubleshooting.md) for your specific issue
2. Enable verbose mode: `codeatlas parse --path ./project --verbose`
3. Review error messages in output metadata

## Common Use Cases

### Parse a Go Project
```bash
codeatlas parse --path ./mygoproject --language go --output go-ast.json
```
**Documentation**: [Complete Documentation - Example 1](./cli-parse-command.md#example-1-parse-go-repository)

### Parse with Custom Ignore Patterns
```bash
codeatlas parse --path ./project \
  --ignore-pattern "*.test.js" \
  --ignore-pattern "*.spec.ts"
```
**Documentation**: [Complete Documentation - Example 2](./cli-parse-command.md#example-2-parse-with-custom-ignore-patterns)

### Parse with Semantic Enhancement
```bash
export CODEATLAS_LLM_API_KEY=sk-your-key
codeatlas parse --path ./project --semantic
```
**Documentation**: 
- [Complete Documentation - Example 5](./cli-parse-command.md#example-5-parse-with-semantic-enhancement)
- [Environment Variables](./parse-environment-variables.md)

### Debug Parsing Issues
```bash
codeatlas parse --path ./project --verbose 2>&1 | tee debug.log
```
**Documentation**: [Troubleshooting Guide](./parse-troubleshooting.md)

### Process Output with jq
```bash
codeatlas parse --path ./project | jq '.files[] | select(.language == "go")'
```
**Documentation**: [Quick Reference - Output Processing](./parse-command-quick-reference.md#output-processing-with-jq)

## Documentation Maintenance

### Adding New Content
When adding new features or documentation:
1. Update the relevant documentation file
2. Add examples to [Quick Reference](./parse-command-quick-reference.md)
3. Add troubleshooting entries if applicable
4. Update this index
5. Update [docs/README.md](./README.md)

### Documentation Standards
- Use clear, concise language
- Include practical examples
- Provide troubleshooting steps
- Cross-reference related documentation
- Keep examples up-to-date with code changes

## Related Documentation

- [Main README](../README.md) - Project overview
- [Testing Documentation](./testing/testing-coverage.md) - Test coverage and guidelines
- [Schema Documentation](./schema.md) - Database schema
- [Error Handling](./error-handling-implementation.md) - Error handling patterns

## Support

- **GitHub Issues**: Report bugs and request features
- **GitHub Discussions**: Ask questions and share ideas
- **Documentation**: This directory contains all guides

## Version

This documentation is for CodeAtlas CLI version 1.0.0.

Last updated: 2025-10-12
