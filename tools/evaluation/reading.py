"""The paired reading of a contrast: the 95 % interval of the paired difference, and what it allows to say.

The suite design (section D, "Reading") pairs the two variants' launches of the
same task, step and repetition, and reads the 95 % t interval of their
differences, oriented so that a positive difference favors the variant over
its baseline:

  better                                 the interval lies above zero
  worse                                  the interval lies below zero
  a meaningful advantage is excluded     it lies below the minimum meaningful difference
  undecided                              otherwise, or with fewer than two pairs

and concept 2.0 §15.5, "at least as trustworthy", reads as non-inferiority:
the interval's lower end lies above minus the minimum meaningful difference.
The minimum meaningful difference is the owner's, from the series protocol.
"""
import math
import statistics

LEVEL = 0.95


def _beta_fraction(a, b, x):
    """The continued fraction of the regularized incomplete beta function (modified Lentz)."""
    tiny, c, d = 1e-300, 1.0, 1.0 - (a + b) * x / (a + 1.0)
    d = 1.0 / (d if abs(d) > tiny else tiny)
    h = d
    for m in range(1, 300):
        for numerator in (m * (b - m) * x / ((a + 2 * m - 1) * (a + 2 * m)),
                          -(a + m) * (a + b + m) * x / ((a + 2 * m) * (a + 2 * m + 1))):
            d = 1.0 + numerator * d
            d = 1.0 / (d if abs(d) > tiny else tiny)
            c = 1.0 + numerator / c
            c = c if abs(c) > tiny else tiny
            h *= d * c
        if abs(d * c - 1.0) < 1e-15:
            break
    return h


def incomplete_beta(a, b, x):
    if x <= 0:
        return 0.0
    if x >= 1:
        return 1.0
    front = math.exp(math.lgamma(a + b) - math.lgamma(a) - math.lgamma(b) + a * math.log(x) + b * math.log(1 - x))
    if x < (a + 1) / (a + b + 2):
        return front * _beta_fraction(a, b, x) / a
    return 1.0 - front * _beta_fraction(b, a, 1 - x) / b


def t_cdf(t, df):
    tail = 0.5 * incomplete_beta(df / 2, 0.5, df / (df + t * t))
    return 1 - tail if t > 0 else tail


def t_quantile(p, df):
    """The p quantile of Student's t with df degrees of freedom, by bisection."""
    low, high = -1e3, 1e3
    for _ in range(200):
        mid = (low + high) / 2
        if t_cdf(mid, df) < p:
            low = mid
        else:
            high = mid
    return (low + high) / 2


def paired(differences, minimum):
    """The interval and the readings of paired differences (positive favors the variant). Without a minimum meaningful
    difference (None) the reading is descriptive: better, worse or undecided, and no claim of at least as good."""
    n = len(differences)
    if n < 2:
        return {"pairs": n, "mean": differences[0] if n else None, "low": None, "high": None,
                "reading": "undecided (fewer than two pairs)", "atLeastAsGood": None}
    mean = statistics.fmean(differences)
    half = t_quantile(1 - (1 - LEVEL) / 2, n - 1) * statistics.stdev(differences) / math.sqrt(n)
    low, high = mean - half, mean + half
    if low > 0:
        reading = "better"
    elif high < 0:
        reading = "worse"
    elif minimum is not None and high < minimum:
        reading = "a meaningful advantage is excluded"
    else:
        reading = "undecided"
    return {"pairs": n, "mean": mean, "low": low, "high": high, "reading": reading,
            "atLeastAsGood": None if minimum is None else low > -minimum}
