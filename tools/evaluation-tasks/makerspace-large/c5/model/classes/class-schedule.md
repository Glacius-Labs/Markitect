---
kind: std/Rule
internal: true
---
# Class schedule

A [[classes/class]] is `scheduled` until it is cancelled, and a cancelled class stays cancelled. A class has no capacity of its own and no opening hours, and it is not moved: to change it, cancel it and schedule another.

A scheduled class keeps its room: scheduling it makes a hold of the room for `[start, end)` with origin `class` and purpose `class ` followed by the class identifier, and cancelling it releases that hold ([[rooms/room-holds]]). A cancelled class holds nothing.

An instructor teaches at most one scheduled class at a time: the scheduled classes of one instructor never overlap ([[time]]). Classes of different instructors may overlap, in different rooms.
