# Documentation Index

This directory contains comprehensive documentation for the go-query library.

## Core Documentation

- **[QUERY_SYNTAX.md](QUERY_SYNTAX.md)** - Complete query language reference
  - Google-style bare search
  - All operators and syntax
  - Complex expressions
  - Real-world examples

- **[CONFIGURATION.md](CONFIGURATION.md)** - Configuration guide
  - Executor options
  - Default search field
  - Parser cache setup
  - Field restrictions
  - Database-specific settings

- **[EXAMPLES.md](EXAMPLES.md)** - Real-world usage examples
  - E-commerce search
  - User management
  - Content search
  - API endpoints
  - Testing examples

- **[PERFORMANCE.md](PERFORMANCE.md)** - Performance optimization
  - Parser cache setup
  - Database indexing
  - Query optimization
  - Best practices

- **[FEATURES.md](FEATURES.md)** - Advanced features guide
  - Parser cache
  - Count method ⭐ **New**
  - Map support
  - Dynamic data sources
  - Custom field getter
  - Query options (pagination, sorting, cursors)
  - REGEX support
  - Unicode handling
  - Field restriction

- **[TESTING.md](TESTING.md)** - Complete testing guide
  - Setup instructions for all executors
  - MongoDB testing with Podman
  - Test coverage statistics
  - Edge cases covered
  - Troubleshooting guide

- **[ERROR_HANDLING.md](ERROR_HANDLING.md)** - Error handling system
  - Public error variables
  - Error types and patterns
  - HTTP status code mapping
  - Usage examples
  - Migration guide

- **[SECURITY.md](SECURITY.md)** - Security features
  - SQL injection protection
  - Field restriction
  - Attack examples (all blocked)
  - Security best practices
  - Code review checklist

## Quick Links

- **Main README**: See `/README.md` in project root
- **Executor READMEs**: See `executors/{memory,gorm,mongodb}/README.md`

## Documentation Principles

- ✅ Concise and practical
- ✅ Examples for every feature
- ✅ Best practices included
- ✅ Troubleshooting guides
- ✅ Migration notes where applicable

---

**Generated**: AI (Claude Sonnet 4.5 and Cursor Auto)
**License**: Apache 2.0
