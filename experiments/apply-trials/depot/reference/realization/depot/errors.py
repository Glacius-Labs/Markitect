"""The refusal that every part raises for invalid input or a violated rule."""


class Refusal(Exception):
    """A command is refused; its message becomes the error object's text."""
