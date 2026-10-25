"""The hidden case set of a step: W0 and every change up to it, cumulatively (requirements README, "Reading the steps")."""

from __future__ import annotations

from typing import Any

import behavior
import invariants
import probes

CHECKS = invariants.CHECKS


def build_cases(step: str) -> list[dict[str, Any]]:
    return behavior.build_behavior(step) + probes.build_probes(step) + invariants.scenario(step)
