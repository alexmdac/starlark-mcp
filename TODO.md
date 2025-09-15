# Starlark MCP Server - TODO List
*Feature requests from Claude after testing the server*

## Priority Features Requested by Claude

### 1. **Return Top-Level Variables** 
**Claude's Request:** Return final variable state after execution alongside existing print output.

**Implementation:**
```json
{
  "output": "=== FRACTAL PATTERNS ===\n*****\n  ***\n   *",
  "variables": {
    "results": [[0, 100, 20], [1, 95, 22]],
    "primes": [2, 3, 5, 7, 11],
    "final_population": 42.5
  }
}
```

**Claude's Rationale:** "This would be transformative - `print()` is perfect for ASCII art, fractals, and formatted displays, while variables capture the underlying computed data for further analysis. Best of both worlds!"

---

### 2. **Execution Metadata** *(Optional)*
**Claude's Request:** Return runtime statistics and execution information.

**Implementation:**
```json
{
  "output": "...",
  "variables": {...},
  "metadata": {
    "execution_time_ms": 45,
    "lines_executed": 120,
    "status": "completed"
  }
}
```

**Claude's Rationale:** "Nice professional touch that helps with performance awareness."

---

### 3. **Safe Standard Library Subset** *(Optional)*
**Claude's Request:** Add basic mathematical functions commonly needed for computational work.

**Suggested additions:**
- `math.sqrt()`, `math.sin()`, `math.cos()`
- `math.pi`, `math.e` constants
- Enhanced statistical functions

**Claude's Rationale:** "Would reduce need for custom mathematical implementations like my `sqrt_approx()` function."