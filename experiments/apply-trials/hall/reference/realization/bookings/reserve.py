"""The reserve that the bookings part chooses: seats of every room held back from booking."""

RESERVE = 2


def bookable(seats):
    """The bookable seats of a room with SEATS seats."""
    return seats - RESERVE
