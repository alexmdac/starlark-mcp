# prelude.star — symbols defined here are predeclared in LLM-generated programs.

load("math", _round = "round", _pow = "pow")

def round(x, ndigits = None):
    """Round x to ndigits decimal places (default: nearest int)."""
    if ndigits == None:
        return int(_round(x))
    mul = _pow(10, ndigits)
    return _round(x * mul) / mul
