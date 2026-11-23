# Building blocks

The building blocks the system is built from. One [component](../../../packages/software/component.md) per document: what it is responsible for, with links to the concepts and operations of the product it realizes, and what it uses. Link each from the section Components of the architecture's README.

Name a building block by what it does. Say what it depends on, and keep the dependencies going one way: when two blocks need each other, one of them is cut wrong. A block that everything else depends on is the core; write down what it knows and what it never knows.
