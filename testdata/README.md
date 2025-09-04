# Test Data

This directory contains workflow JSON files used for testing and examples.

## Workflow Files

- `file_organizer_workflow.json` - File organization automation
- `file_processor_workflow.json` - Document processing workflow  
- `golang_monitor.json` - Go repository monitoring
- `iphone_price_check.json` - Product price checking
- `llm_analysis_workflow.json` - LLM-powered analysis workflow
- `llm_test_workflow.json` - Simple LLM test workflow
- `price_monitor_workflow.json` - Price monitoring automation
- `simple_download_workflow.json` - File download workflow
- `website_monitor_workflow.json` - Website change monitoring

## Usage

These files are used by:
- Test scripts (`scripts/run_llm_workflow.sh`)
- Example documentation (`examples/practical_workflow.md`)
- Unit tests and integration tests
- Development and debugging workflows

## Format

All files follow the RAGO workflow JSON specification with:
- `steps[]` - Array of workflow steps
- `id` - Unique step identifier
- `tool` - MCP tool to execute
- `inputs` - Tool input parameters
- `outputs` - Variable mapping for step results