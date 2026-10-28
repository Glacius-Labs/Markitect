---
kind: Kind
---
# Project

A Project Definition is the project's anchor: what the project is, for whom, and what it deliberately does not do. It sits in the model's root folder, the Root Package, so that every Manager reads it.

Write it as the opening page of good documentation:

- what the project is and the problem it solves, in a few sentences;
- who uses it and through which interfaces;
- the main parts, each a Package, linked by path (`[[members]]`);
- what is out of scope.

There is usually one Project Definition. Project-wide [[std/Rule|Rules]], layout conventions and the architecture are separate Definitions beside it in the Root Package.
