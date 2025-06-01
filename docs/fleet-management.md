# Fleet Management Tools

This document describes the Fleet management tools available in the Elasticsearch CLI, which allow you to interact with Kibana Fleet APIs from the command line.

## Overview

Kibana Fleet is a centralized management interface for Elastic Agents. It allows you to add integrations to collect data from your systems and services, manage agent policies, and monitor the status of your agents. The Elasticsearch CLI now provides command-line tools to interact with Fleet APIs, making it easier to automate Fleet management tasks.

## Available Commands

The following Fleet management commands are available:

| Command | Description |
|---------|-------------|
| `kb_fleet_policies` | List all agent policies from Kibana Fleet |
| `kb_fleet_tokens` | List all enrollment tokens from Kibana Fleet |
| `kb_fleet_integrations` | List all package policies (integrations) from Kibana Fleet |

## Configuration

Fleet management commands use the same configuration options as other Kibana commands. You can configure the connection to Kibana using the following methods:

1. **Command-line flags**:
   ```
   --kb-addresses=https://kibana:5601
   --kb-username=elastic
   --kb-password=changeme
   --kb-ca-cert=/path/to/ca.crt
   --kb-insecure=false
   ```

2. **Environment variables**:
   ```
   ESCTL_KIBANA_ADDRESSES=https://kibana:5601
   ESCTL_KIBANA_USERNAME=elastic
   ESCTL_KIBANA_PASSWORD=changeme
   ESCTL_KIBANA_CA_CERT=/path/to/ca.crt
   ESCTL_KIBANA_INSECURE=false
   ```

3. **Configuration file**:
   ```yaml
   kibana:
     addresses:
       - https://kibana:5601
     username: elastic
     password: changeme
     ca_cert: /path/to/ca.crt
     insecure: false
   ```

## Output Formats

All Fleet management commands support multiple output formats:

- `fancy` (default): A visually appealing table format with customizable styles
- `plain`: A simple text table format
- `json`: JSON format for machine processing
- `csv`: CSV format for importing into spreadsheets

You can specify the output format using the `--format` or `-f` flag:

```
kb_fleet_policies --format=json
```

For the `fancy` format, you can also specify a style using the `--style` flag:

```
kb_fleet_policies --style=blue
```

Available styles: `dark` (default), `light`, `bright`, `blue`, `double`.

## Examples

### List Agent Policies

```
kb_fleet_policies
```

Example output:
```
┌──────────────────────────────────┬───────────────┬───────────┬────────┬──────────┬─────────────────────────┐
│ ID                               │ NAME          │ NAMESPACE │ STATUS │ REVISION │ UPDATED AT              │
├──────────────────────────────────┼───────────────┼───────────┼────────┼──────────┼─────────────────────────┤
│ 2b820230-4b54-11ed-b107-4bfe66d7 │ Agent policy  │ default   │ active │ 1        │ 2022-10-14T00:07:19.76 │
│                                  │ 1             │           │        │          │ 3Z                      │
└──────────────────────────────────┴───────────────┴───────────┴────────┴──────────┴─────────────────────────┘
```

### List Enrollment Tokens

```
kb_fleet_tokens
```

Example output:
```
┌──────────────────────────────────┬───────────────────────────────────────┬──────────────────────────────────┬────────┬─────────────────────────┐
│ ID                               │ NAME                                  │ POLICY ID                        │ ACTIVE │ CREATED AT              │
├──────────────────────────────────┼───────────────────────────────────────┼──────────────────────────────────┼────────┼─────────────────────────┤
│ 39703af4-5945-4232-90ae-31612145 │ Default (39703af4-5945-4232-90ae-3161 │ 2b820230-4b54-11ed-b107-4bfe66d7 │ Yes    │ 2022-10-14T00:07:21.42 │
│ 12fa                             │ 214512fa)                             │ 59e4                             │        │ 0Z                      │
└──────────────────────────────────┴───────────────────────────────────────┴──────────────────────────────────┴────────┴─────────────────────────┘
```

### List Package Policies (Integrations)

```
kb_fleet_integrations
```

Example output:
```
┌──────────────────────────────────┬────────────────┬─────────┬─────────┬──────────────────────────────────┐
│ ID                               │ NAME           │ PACKAGE │ VERSION │ POLICY ID                        │
├──────────────────────────────────┼────────────────┼─────────┼─────────┼──────────────────────────────────┤
│ 4f8c2a40-4b54-11ed-b107-4bfe66d7 │ nginx-demo-123 │ nginx   │ 1.5.0   │ 2b820230-4b54-11ed-b107-4bfe66d7 │
│ 59e4                             │                │         │         │ 59e4                             │
└──────────────────────────────────┴────────────────┴─────────┴─────────┴──────────────────────────────────┘
```

## Troubleshooting

### Authentication Issues

If you encounter authentication issues, ensure that:

1. You have provided the correct Kibana credentials
2. The user has sufficient permissions to access Fleet APIs
3. If using API keys, the API key has the required privileges

### Connection Issues

If you cannot connect to Kibana:

1. Verify that the Kibana URL is correct and accessible
2. Check if TLS certificates are properly configured
3. If using self-signed certificates, use the `--kb-ca-cert` flag to provide the CA certificate or `--kb-insecure` to skip certificate validation (not recommended for production)

### API Errors

If you receive API errors:

1. Ensure that Fleet is enabled in your Kibana instance
2. Check that your Kibana version supports the Fleet APIs being used
3. Verify that the user has the necessary permissions to access Fleet APIs

## Future Enhancements

Future versions of the Elasticsearch CLI may include additional Fleet management features:

- Creating and updating agent policies
- Creating and updating package policies
- Managing Elastic Agents
- Retrieving agent status information

## References

- [Kibana Fleet APIs Documentation](https://www.elastic.co/docs/reference/fleet/fleet-api-docs)
- [Elastic Agent Documentation](https://www.elastic.co/guide/en/fleet/current/elastic-agent-installation.html)
