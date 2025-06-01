# Fancy Output Format for Elasticsearch CLI

## Overview

The Elasticsearch CLI tools now support a modern, visually appealing "fancy" output format using the `go-pretty/v6` library. This format provides rich table styling with features like colored headers, borders, and automatic column sizing for improved readability of Elasticsearch data.

## Features

- **Styled Tables**: Colorful, well-formatted tables with clear headers and data separation
- **Multiple Style Options**: Choose from different visual styles to match your terminal preferences
- **Automatic Column Sizing**: Columns automatically adjust based on content
- **Pagination**: Large result sets are paginated for better navigation
- **Summary Information**: Includes record count and other metadata

## Usage

### Command Line

To use the fancy output format, use the `--format` flag with the value `fancy`:

```bash
es_nodes list --format fancy
```

### Style Customization

You can customize the table style using the `--style` flag:

```bash
es_nodes list --format fancy --style light
```

Available styles:

- **dark** (default): Dark background with colored headers and borders
- **light**: Light styling suitable for light terminal backgrounds
- **bright**: Bright colors with green headers
- **blue**: Blue and white color scheme
- **double**: Double-line borders for a classic look

## Configuration

### Default Settings

The default output format is set to `fancy` in the configuration file. You can change this in your `config.yaml`:

```yaml
output:
  format: "fancy"  # fancy, plain, json, csv
  style: "dark"    # dark, light, bright, blue, double
```

### Environment Variables

You can also set the output format and style using environment variables:

```bash
export ESCTL_OUTPUT_FORMAT=fancy
export ESCTL_OUTPUT_STYLE=blue
```

## Examples

### Node List with Dark Style (Default)
```
es_nodes list --format fancy
```

### Index Information with Light Style
```
es_indices list --format fancy --style light
```

### Shard Allocation with Bright Style
```
es_shards list --format fancy --style bright
```

## Comparison with Other Formats

The Elasticsearch CLI tools support multiple output formats:

- **fancy**: Rich, styled tables with colors and formatting (default)
- **plain**: Simple ASCII tables without colors
- **json**: Raw JSON output for programmatic consumption
- **csv**: CSV format for importing into spreadsheets

## Technical Implementation

The fancy output format is implemented using the `github.com/jedib0t/go-pretty/v6` library, which provides robust table rendering capabilities with extensive styling options. The implementation includes:

- Header formatting with title case and color highlighting
- Configurable border styles based on the selected style
- Automatic column width adjustment
- Row and column separation for improved readability
- Pagination for large result sets

## Troubleshooting

If colors don't display correctly in your terminal:
- Ensure your terminal supports ANSI color codes
- Try a different style option that might work better with your terminal
- Fall back to the `plain` format if needed
