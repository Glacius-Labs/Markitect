# Class

A class of [Areas](area.md): what this kind of Area is and what holds for it, in a few lines. Features and stories below an Area inherit the rules of its class; an Area may add stricter ones. Take it to group Areas that follow the same rules. A class is optional. It says who usually plans there, never who may: breaking a class's rule is a deviation with a level, not a block. The Architect explains the class when briefing a person, and the Reviewer checks against its rules.

## Purpose
The one goal of this kind of Area. Example: `A variant of entering a loan.`

## Connects to
Where it connects and what it may rely on; `nothing` when everything else connects to it. Example: `The catalog's interface for copies.`

## Never
What it never does, in short sentences. Example: `Changes a member's data.`

## Usually planned by
The role that usually plans features and stories in Areas of this class, as a link, or `the owner of the Area`. Example: `[Admin](../roles/admin.md)`.
