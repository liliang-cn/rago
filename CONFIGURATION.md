# RAGO Configuration Guide

This directory contains several configuration examples for different use cases:

## Available Configuration Files

### 1. `config-complete.toml` - Complete Configuration
A comprehensive configuration file showing all available options and their descriptions. Use this as a reference for all possible settings.

**Features:**
- Full provider system configuration
- All tool configurations
- Detailed comments explaining each option
- Both Ollama and OpenAI provider examples

### 2. `config-openai.toml` - Pure OpenAI
Use OpenAI for both LLM and embeddings.

**Best for:**
- Users who prefer OpenAI's quality
- Production deployments with API budget
- When you need the best performance

**Requirements:**
- OpenAI API key
- Internet connection

### 3. `config-mixed.toml` - Hybrid Configuration
Use OpenAI for LLM (better quality) and Ollama for embeddings (cost-effective).

**Best for:**
- Balancing quality and cost
- Users who want high-quality responses but lower embedding costs
- Hybrid cloud/local deployments

**Requirements:**
- OpenAI API key for LLM
- Local Ollama installation for embeddings

### 4. `config-local-openai.toml` - Local OpenAI-Compatible
For use with local OpenAI-compatible services like vLLM, LocalAI, etc.

**Best for:**
- Fully local deployments
- Privacy-sensitive environments
- Custom model hosting

**Requirements:**
- Local OpenAI-compatible service (vLLM, LocalAI, etc.)

## Quick Setup

1. Choose a configuration file based on your needs
2. Copy it to your project root as `config.toml`
3. Update the configuration values:
   - For OpenAI: Set your `api_key`
   - For Ollama: Ensure `base_url` points to your Ollama instance
   - For local services: Update `base_url` and model names
4. Adjust `vector_dim` to match your embedding model:
   - Ollama nomic-embed-text: `768`
   - OpenAI text-embedding-3-small: `1536`
   - OpenAI text-embedding-3-large: `3072`

## Environment Variables

You can override configuration values using environment variables:

```bash
export RAGO_PROVIDERS_DEFAULT_LLM=openai
export RAGO_PROVIDERS_DEFAULT_EMBEDDER=ollama
export RAGO_OPENAI_API_KEY=sk-your-key
export RAGO_OLLAMA_BASE_URL=http://localhost:11434
```

## Migration from Legacy Configuration

If you have an existing `config.toml` with the old `[ollama]` section, it will continue to work. To migrate to the new provider system:

1. Add the `[providers]` section
2. Move your Ollama configuration to `[providers.ollama]`
3. Set the default providers
4. Optionally add OpenAI configuration

The old configuration will be used as a fallback if no provider configuration is found.