"""Versioned interfaces for driving independent Playground methods."""

from .contract import Adapter, AdapterError, validate_descriptor, validate_result

__all__ = ["Adapter", "AdapterError", "validate_descriptor", "validate_result"]
