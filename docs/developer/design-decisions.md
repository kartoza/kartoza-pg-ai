# Design Decisions

This document explains key architectural and design decisions.

## Rule-Based Query Engine vs LLM

### Decision

Use a rule-based query engine initially with pattern matching.

### Rationale

1. **Simplicity**: No external dependencies or model files
2. **Speed**: Instant query generation without model inference
3. **Predictability**: Deterministic output for the same input
4. **Privacy**: All processing happens locally

### Future Direction

The architecture supports replacing the rule-based engine with:

- llama.cpp integration for local LLM inference
- External API integration (optional)
- Hybrid approach with fallback

## Schema Caching

### Decision

Cache database schemas locally with configurable TTL.

### Rationale

1. **Startup Speed**: Avoid harvesting on every launch
2. **Network**: Reduce database round-trips
3. **User Experience**: Near-instant connection to known databases

### Implementation

- Cache stored in `~/.config/kartoza-pg-ai/config.json`
- TTL-based invalidation (default 24 hours)
- Manual refresh option

## pg_service.conf Integration

### Decision

Use existing PostgreSQL service file instead of custom configuration.

### Rationale

1. **Familiarity**: Standard PostgreSQL mechanism
2. **Reusability**: Works with psql and other tools
3. **Security**: Credentials stay in standard location
4. **Simplicity**: No duplicate configuration

## Bubble Tea for TUI

### Decision

Use Bubble Tea and the Charm ecosystem.

### Rationale

1. **Modern**: Clean Elm-architecture pattern
2. **Active**: Well-maintained with regular updates
3. **Ecosystem**: Lipgloss, Bubbles for styling and components
4. **Quality**: Production-ready with good documentation

## DRY Header System

### Decision

Single `RenderHeader()` function used across all screens.

### Rationale

1. **Consistency**: Identical appearance everywhere
2. **Maintainability**: Single point of change
3. **Global State**: Status bar reflects app-wide state

## Static Binaries

### Decision

Build with CGO_ENABLED=0 for fully static binaries.

### Rationale

1. **Portability**: No runtime dependencies
2. **Distribution**: Single file deployment
3. **Compatibility**: Works on any compatible OS version
