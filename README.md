# gy - YAML Path Extractor

A fast, lightweight command-line tool for extracting, exploring, and navigating YAML documents. Think `jq` for YAML, built in Go with minimal dependencies.

## Features

- 🎯 **Path-based extraction** - Navigate YAML with simple dot notation
- 📋 **List mode** - Explore document structure interactively
- 🔍 **Trim mode** - Extract just the data you need
- 📥 **Pipe-friendly** - Works with files or stdin
- ⚡ **Fast** - Single binary, minimal overhead

## Installation

```bash
git clone https://github.com/tsettle/gy
cd gy
go build gy.go
sudo cp gy /usr/local/bin/
```

Or install directly:
```bash
go install github.com/tsettle/gy@latest
```

## Quick Start

Given this `config.yml`:
```yaml
database:
  host: localhost
  port: 5432
  credentials:
    user: admin
    password: secret
services:
  - name: web
    port: 8080
  - name: api
    port: 3000
```

### Basic Extraction

```bash
# Extract with full path preserved
$ gy 'database.host' config.yml
database:
  host: localhost

# Extract just the value
$ gy -t 'database.host' config.yml
localhost

# Navigate into nested structures
$ gy 'database.credentials' config.yml
database:
  credentials:
    user: admin
    password: secret
```

### Array Access

```bash
# Access array elements by index
$ gy 'services[0]' config.yml
services:
  - name: web
    port: 8080

# Navigate into array elements
$ gy -t 'services[1].name' config.yml
api

# Trim mode works great with arrays
$ gy -t 'services[0].port' config.yml
8080
```

### Discovery Mode

```bash
# List top-level keys
$ gy -l '.' config.yml
database
services

# List with depth
$ gy -l --depth 2 'database' config.yml
database
  host
  port
  credentials
    user
    password

# Explore array contents
$ gy -l 'services[0]' config.yml
name
port
```

### Piping and Composition

```bash
# Read from stdin
$ cat config.yml | gy 'database.credentials'

# Chain with other tools
$ gy -t 'services[0].name' config.yml | tr '[:lower:]' '[:upper:]'
WEB

# Use in scripts
$ DB_HOST=$(gy -t 'database.host' config.yml)
$ echo "Connecting to $DB_HOST"
```

## Usage

```
gy [OPTIONS] <path> [filename]
```

### Options

| Flag | Description |
|------|-------------|
| `-t, --trim` | Return only the matched node (no path wrapping) |
| `-l, --list` | List all keys/indices under the path |
| `--depth N` | Control listing depth (default: 1, use 0 for unlimited) |

### Path Syntax

- **Dot notation**: `path.to.key`
- **Array indexing**: `path.to.array[0]`
- **Combined**: `users[0].profile.email`
- **Root**: `.` or leave empty to reference the entire document

## Common Patterns

### Configuration Management

```bash
# Extract database config for deployment
gy 'database' prod.yml > db-config.yml

# Get specific credentials
export DB_PASS=$(gy -t 'database.credentials.password' config.yml)
```

### Exploration

```bash
# Quick overview of document structure
gy -l --depth 3 '.' large-config.yml

# Find all available modules
gy -l 'modules' snmp.yml
```

### Validation

```bash
# Check if a path exists
if gy 'database.host' config.yml > /dev/null 2>&1; then
    echo "Database configured"
fi
```

### Data Extraction

```bash
# Extract multiple configs in a script
for service in web api worker; do
    gy "services.$service" config.yml > "$service-config.yml"
done
```

## Real-World Examples

### SNMP Configuration

```bash
# List all monitoring modules
gy -l 'modules' snmp.yml

# Extract specific MIB walks
gy --trim 'modules.if_mib.walk' snmp.yml > if_mib_walk.yml

# Get first OID from a walk
gy -t 'modules.if_mib.walk[0]' snmp.yml
```

### Kubernetes Manifests

```bash
# Extract container specs
gy 'spec.template.spec.containers[0]' deployment.yml

# List all environment variables
gy -l 'spec.template.spec.containers[0].env' deployment.yml
```

### CI/CD Pipelines

```bash
# Extract job definitions
gy '.github.workflows.build.jobs' .github/workflows/ci.yml

# Get specific job steps
gy -t '.github.workflows.build.jobs.test.steps[0]' ci.yml
```

## Roadmap

- [ ] **Wildcard support** - `gy 'users[*].name'` to extract from all array items
- [ ] **Glob patterns** - `gy 'services.*.port'` for flexible matching  
- [ ] **Merge functionality** - `gy --merge target.yml 'path.to.data' source.yml`
- [ ] **Flat list mode** - Output full paths on single lines for grep compatibility
- [ ] **Multiple patterns** - `gy 'path1,path2,path3'`
- [ ] **JSON output** - `gy --json` for cross-format workflows
- [ ] **Named lists** - `gy '@name:Deploy web application stack' ansible.yml`
- [ ] **Key/Value** - Return any paths matching a key and/or value.

## Contributing

Contributions welcome! Please feel free to submit issues or pull requests.

## License

MIT License - see LICENSE file for details

## Similar Tools

- [yq](https://github.com/mikefarah/yq) - Feature-rich YAML processor (uses YAML→JSON→YAML conversion)
- [jq](https://github.com/stedolan/jq) - JSON processor that inspired this tool
- [dasel](https://github.com/TomWright/dasel) - Unified selector for JSON/YAML/TOML/XML

**Why gy?** Because `yq` doesn't preserve paths in output without bracket gymnastics, converts YAML→JSON→YAML (destroying types, comments, and anchors), and sometimes you just need to extract a chunk from a 1.5MB YAML file and merge it into another without your `1:` keys becoming `"1":` strings. Built for the 90% use case: extract, preserve structure, preserve types, move on with your life.
