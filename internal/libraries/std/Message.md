---
kind: Kind
fields:
  To: {required: true}
  Sent when: {required: true, links: [std/Reaction, std/Operation, std/Outcome]}
  Holds: {required: true}
parts:
---
# Message

A Message Definition states one message the system sends to a person by itself, such as an e-mail or a text message. It lives under `messages/` of the Package of its interface. **To:** who receives it. **Sent when:** the [[std/Reaction]], the [[std/Operation]] or the [[std/Outcome]] that makes it due. **Holds:** what it says, named by what the Operations return; the wording is free unless stated.

Verified when it is sent when stated, to whom stated, and holds what is stated.
