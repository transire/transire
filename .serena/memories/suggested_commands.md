# Transire Suggested Commands

## Project Commands (Design Phase)

Since this is currently a design-only repository, most commands are conceptual or for examining the design.

### File Operations (Darwin/macOS)
- `ls -la` - List files and directories with details
- `find . -name "*.go"` - Find Go files in the project
- `find . -name "*.yaml"` - Find configuration files
- `grep -r "pattern" .` - Search for text patterns

### Git Operations
- `git status` - Check repository status
- `git diff` - View changes
- `git log --oneline` - View commit history

### Design Review Commands
- `cat DESIGN.md` - View main design specification
- `cat DESIGN_PROMPT.md` - View design requirements
- `cat examples/simple-api/main.go` - View example application
- `cat pkg/transire/interfaces.go` - View core interface definitions

### Future CLI Commands (When Implemented)
```bash
# Create new project
transire init my-app

# Run locally with simulated cloud services  
transire run

# Build and deploy to AWS
transire build
transire deploy
```

### Development Workflow (Future)
1. Read design documents
2. Explore examples directory
3. Review package interfaces
4. Validate against DESIGN_PROMPT.md requirements