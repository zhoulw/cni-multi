# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`cni-multi` is a Go project related to Kubernetes CNI (Container Network Interface) operations, specifically CNI switching/migration (e.g., Calico to FlexNet CNI). See `doc/` for detailed design and operations documentation.

## Build & Run

```bash
go run main.go
```

## Architecture

- **Module**: `cni-multi` (Go 1.24)
- This project is in early-stage scaffolding; `main.go` contains only an empty `main()` function.
- Design documents are in the `doc/` directory (`.docx` format).
