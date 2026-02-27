# Starlark Built-in Functions

This document describes the built-in functions available in the Starlark MCP server.

## Math Module

Mathematical functions and constants. Import with:

```python
load("math", "sqrt", "pow", "sin", "pi")
```

All math functions accept both `int` and `float` arguments.

### Constants

| Name | Description |
|------|-------------|
| `pi` | The ratio of a circle's circumference to its diameter, approximately 3.14159. |
| `e`  | The base of natural logarithms, approximately 2.71828. |

### Rounding and Absolute Value

| Function | Description |
|----------|-------------|
| `ceil(x)` | Returns the smallest integer greater than or equal to x. |
| `floor(x)` | Returns the largest integer less than or equal to x. |
| `round(x)` | Returns the nearest integer, rounding half away from zero. |
| `fabs(x)` | Returns the absolute value of x as a float. |

### Arithmetic

| Function | Description |
|----------|-------------|
| `pow(x, y)` | Returns x raised to the power y. |
| `sqrt(x)` | Returns the square root of x. |
| `exp(x)` | Returns e raised to the power x. |
| `log(x[, base])` | Returns the logarithm of x in the given base (natural log by default). |
| `mod(x, y)` | Returns the floating-point remainder of x/y. |
| `remainder(x, y)` | Returns the IEEE 754 floating-point remainder of x/y. |
| `copysign(x, y)` | Returns a value with the magnitude of x and the sign of y. |
| `gamma(x)` | Returns the Gamma function of x. |

### Trigonometry

| Function | Description |
|----------|-------------|
| `sin(x)` | Returns the sine of x (in radians). |
| `cos(x)` | Returns the cosine of x (in radians). |
| `tan(x)` | Returns the tangent of x (in radians). |
| `asin(x)` | Returns the arc sine of x, in radians. |
| `acos(x)` | Returns the arc cosine of x, in radians. |
| `atan(x)` | Returns the arc tangent of x, in radians. |
| `atan2(y, x)` | Returns atan(y/x), in radians, using the signs of both arguments to determine the quadrant. |
| `hypot(x, y)` | Returns the Euclidean norm, sqrt(x\*x + y\*y). |
| `degrees(x)` | Converts angle x from radians to degrees. |
| `radians(x)` | Converts angle x from degrees to radians. |

### Hyperbolic Functions

| Function | Description |
|----------|-------------|
| `sinh(x)` | Returns the hyperbolic sine of x. |
| `cosh(x)` | Returns the hyperbolic cosine of x. |
| `tanh(x)` | Returns the hyperbolic tangent of x. |
| `asinh(x)` | Returns the inverse hyperbolic sine of x. |
| `acosh(x)` | Returns the inverse hyperbolic cosine of x. |
| `atanh(x)` | Returns the inverse hyperbolic tangent of x. |

### Examples

```python
load("math", "sqrt", "pow", "sin", "pi", "log", "ceil", "floor")

def main():
    print(sqrt(16))        # 4.0
    print(pow(2, 10))      # 1024.0
    print(sin(pi / 2))     # 1.0
    print(log(100, 10))    # 2.0
    print(ceil(2.3))       # 3
    print(floor(2.7))      # 2

main()
```
