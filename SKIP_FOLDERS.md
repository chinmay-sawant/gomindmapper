# Skip External Dependencies Feature

This document explains how to use the new `--skip-folders` feature in GoMindMapper.

## Overview

When scanning external dependencies with the `--include-external` flag, you might want to skip certain folders or modules to:
- Reduce scanning time
- Avoid processing standard library modules (like `golang.org`)
- Exclude specific vendor dependencies
- Focus on relevant external dependencies only

## Usage

### CLI

```bash
# Skip golang.org modules
./gomindmapper -include-external -skip-folders "golang.org"

# Skip multiple patterns
./gomindmapper -include-external -skip-folders "golang.org,google.golang.org,github.com/go-playground"

# Skip with custom path
./gomindmapper -path /path/to/project -include-external -skip-folders "golang.org"
```

### Server

```bash
# Skip golang.org modules
./server -include-external -skip-folders "golang.org"

# Skip multiple patterns
./server -include-external -skip-folders "golang.org,google.golang.org" -addr ":8080"
```

## How It Works

1. **Pattern Matching**: The skip patterns use simple string containment matching. If a module path contains any of the specified patterns, it will be skipped.

2. **Filtering Order**: 
   - First, all modules are read from `go.mod`
   - Then, modules matching skip patterns are filtered out
   - Finally, only relevant modules (those actually called in your code) are scanned

3. **Example**: If your `go.mod` contains:
   ```
   golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
   google.golang.org/grpc v1.40.0
   github.com/gin-gonic/gin v1.7.4
   ```
   
   Using `-skip-folders "golang.org"` will skip the first two modules and only scan `github.com/gin-gonic/gin`.

## Common Skip Patterns

- `golang.org` - Skip all standard library extended packages
- `google.golang.org` - Skip Google Go packages
- `github.com/golang` - Skip official Go team repositories
- `gopkg.in` - Skip gopkg.in packages
- Specific vendors like `github.com/aws`, `github.com/docker`, etc.

## Benefits

1. **Performance**: Significantly reduces scanning time by skipping large standard library modules
2. **Focus**: Helps focus on application-specific external dependencies
3. **Memory**: Reduces memory usage by not loading unnecessary external functions
4. **Clarity**: Produces cleaner function maps with relevant external dependencies only

## Example Output

```bash
$ ./gomindmapper -include-external -skip-folders "golang.org"
Scanning external modules...
Skipping external dependency folders matching: [golang.org]
Found 32 modules in go.mod
After filtering skip patterns, scanning 23 modules
Scanning module: github.com/gin-gonic/gin@v1.11.0
Found 386 functions in module github.com/gin-gonic/gin
...
Successfully scanned external modules and found 3520 external functions
```

This shows that 9 modules (32 - 23 = 9) were skipped due to containing "golang.org" in their path.