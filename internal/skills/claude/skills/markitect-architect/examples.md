# Example: a lending library

A small Go service that lends the books of a neighborhood library to its members: the staff use it in a browser at the desk, and a member gets an e-mail for each book they borrow. It compiles as written, and the outputs shown are what the tools print. To stay short it states one Operation, borrowing a book, and leaves out much of what the [coverage list](reference.md#coverage) asks of a spec handed over: returning a book, refusing a member or a book the library does not know, how a refusal reaches the person at the desk, most qualities, the technology and how it is run. The spec has Root and four Packages: the areas `domain/members` and `domain/catalog`, the site `interfaces/desk` and the e-mails `interfaces/email`.

## The repository

```text
LICENSE  go.mod
.github/workflows/ci.yml
cmd/library/main.go
internal/members/{member.go, member_test.go}
internal/catalog/{book.go, borrow.go, borrow_test.go}
internal/desk/{pages.go, borrow_test.go}
internal/email/{receipt.go, receipt_test.go}
.markitect/
  manifest.yaml  lock.yaml  ignore  registry.md
  registry/domain/{members.md, catalog.md}
  registry/interfaces/{desk.md, email.md}
  spec/
    project.md
    purpose/goals/know-who-holds-each-book.md
    purpose/stakeholders/staff/{staff, wishes/lend-a-book}.md
    purpose/exclusions/reservations.md
    context/terms/isbn.md
    quality/performance-efficiency/{performance-efficiency, a-loan-within-a-second}.md
    architecture/rules/layout.md
    domain/members/
      members.md
      concepts/member/{member, attributes/member-id, attributes/email, invariants/within-the-loan-limit}.md
    domain/catalog/
      catalog.md
      concepts/book/{book, attributes/isbn}.md
      operations/borrow-book/
        borrow-book.md
        conditions/{book-is-available, member-is-under-the-limit}.md
        outcomes/member-holds-the-book.md
    interfaces/desk/
      desk.md
      pages/borrow/{borrow, actions/borrow}.md
    interfaces/email/
      email.md
      messages/loan-receipt.md
```

`manifest.yaml` names the Library; run `markitect tidy` after changing it:

```yaml
libraries:
  std: 2.0.0
```

## Root: purpose, context, quality and architecture

Root holds what binds the whole project. Its top-level folders sort it by question and are no Packages here: `purpose/` says why the system exists, `context/` what is given from outside, `quality/` how well it must work, `architecture/` how it is built. A folder appears when the project has something to say there.

`spec/project.md` is the project Definition, which opens the spec: what the system is and the problem it removes.

```markdown
---
kind: std/Project
---
# Lending Library

A small service that lends the books of a neighborhood library to its registered members. Today the staff write each loan on a card filed by member, and nobody can say who holds a book without going through the cards.
```

`spec/purpose/goals/know-who-holds-each-book.md` is a Goal: the result the owner wants, stated without how the system gets there. What serves it links it in a **Satisfies:** field; the Goal does not link what serves it.

```markdown
---
kind: std/Goal
---
# Know who holds each book

**Result:** The staff can see at any moment which member holds a book.

**Changes:** A book missing from its shelf is found without going through the loan cards.

**Obstacle:** The loan cards are filed by member, not by book.
```

The staff are a Stakeholder with one Wish, a thing with a part:

`spec/purpose/stakeholders/staff/staff.md`

```markdown
---
kind: std/Stakeholder
---
# Staff

The people who lend the library's books at the desk.
```

`spec/purpose/stakeholders/staff/wishes/lend-a-book.md`

```markdown
---
kind: std/Wish
---
# Lend a book

Lend a book to a member while they wait at the desk.
```

`spec/purpose/exclusions/reservations.md` rules a topic out that a reader could take as included.

```markdown
---
kind: std/Exclusion
---
# Reservations

Reserving a book that another member holds is left out: the member asks at the desk again.
```

`spec/context/terms/isbn.md` is a Term: a word of the world that the system uses and does not manage.

```markdown
---
kind: std/Term
---
# ISBN

An ISBN is the number a publisher gives one edition of a book and prints on it.

**Not:** a number the library gives to one copy.
```

`quality/` holds one Quality per characteristic of ISO/IEC 25010 the project has something to say about, with its requirements as parts. Here it is one, with one requirement and its Measure.

`spec/quality/performance-efficiency/performance-efficiency.md`

```markdown
---
kind: std/Quality
---
# Performance efficiency

Lending a book must not keep the member waiting at the desk.

**Matters to:** [[purpose/stakeholders/staff]].
```

`spec/quality/performance-efficiency/a-loan-within-a-second.md`

```markdown
---
kind: std/Quality
---
# A loan within a second

The staff lend a book while the member waits, and the page answers.

**Measure:** within `1` second for `95` of `100` loans, with `20000` books and `3000` members stored.
```

`spec/architecture/rules/layout.md` is a Rule at Root, so it binds every Package without a link.

```markdown
---
kind: std/Rule
---
# Layout

The code of each Package lives in `internal/<name>/`, where `<name>` is the last part of the Package's path, with its tests beside it as `*_test.go`, and `go test ./...` runs them all. The program starts in `cmd/library/main.go`.
```

The project needs no Kind of its own. A project Kind earns its place when several Definitions share a form that no Library Kind gives.

## Package domain/members

`spec/domain/members/members.md` is the Package's anchor: a Definition of kind `Package` that carries the folder's name makes the folder a Package, and says what it is responsible for and what it keeps to itself. A folder without one, such as `domain/members/concepts/`, is a namespace of `domain/members` and nothing more.

```markdown
---
kind: Package
---
# Members

Who is registered with the library, and how many books one member may hold at once.

It keeps to itself how the library reaches a member.
```

The member is a thing with parts: the folder `concepts/member/` holds `member.md`, which says in one sentence what a member is, and its parts in `attributes/` and `invariants/`, the folders `std/Concept` declares for them. A link to the folder, `[[domain/members/concepts/member]]`, means the member, and `status` lists the member with its parts as one item.

`spec/domain/members/concepts/member/member.md`

```markdown
---
kind: std/Concept
---
# Member

A member is a person registered with the library, who may borrow its books.
```

`spec/domain/members/concepts/member/attributes/member-id.md` is the member's identity. Other Packages name a member by it without relying on the rest of what a member is.

```markdown
---
kind: std/Attribute
---
# Member id

`memberId` is the member's identity: an opaque string the library issues at registration, which never changes and is never reused.
```

`spec/domain/members/concepts/member/attributes/email.md` is `internal: true`: the anchor says the Package keeps it to itself, so only `domain/members` may link to it.

```markdown
---
kind: std/Attribute
internal: true
---
# Email

`email` is the address the library writes to the member at. It holds an `@`, and two addresses that differ only in upper and lower case are the same.
```

`spec/domain/members/concepts/member/invariants/within-the-loan-limit.md` holds for every member at all times, so it is an Invariant of the member and not a Rule, which would hold across Concepts. It states the limit and nothing about what happens when it is reached: the Condition that checks it and the Operation that answers it say that, once.

```markdown
---
kind: std/Invariant
---
# Within the loan limit

A member holds at most `3` books at once.
```

## Package domain/catalog

`domain/catalog` needs members, since a member who borrows must be under the limit, so it depends on `domain/members`; `domain/members` must never depend back.

`spec/domain/catalog/catalog.md`

```markdown
---
kind: Package
---
# Catalog

Which books the library owns, and which member holds each of them.

It keeps to itself how a loan is stored.
```

`spec/domain/catalog/concepts/book/book.md`

```markdown
---
kind: std/Concept
---
# Book

A book is a title the library owns, in one copy.
```

`spec/domain/catalog/concepts/book/attributes/isbn.md`

```markdown
---
kind: std/Attribute
---
# ISBN

`isbn` is the book's [[context/terms/isbn|ISBN]], `13` digits without hyphens.
```

Borrowing is an Operation with parts. `operations/borrow-book/borrow-book.md` says what it achieves and states the fields `markitect doc std/Operation` lists, each on a line of its own: who may call it, what is given, its Conditions in the order they are checked with the code that answers each, what it returns, and what it satisfies. Its Conditions and its Outcome are Definitions of their own, in the folders the Kind declares.

`spec/domain/catalog/operations/borrow-book/borrow-book.md`

```markdown
---
kind: std/Operation
---
# Borrow book

Lends a book to a member. The service has no sign-in, so any caller may.

**Who:** `anyone`.

**Given:** `memberId`, a [[domain/members/concepts/member/attributes/member-id|member identity]], and the [[domain/catalog/concepts/book/attributes/isbn|`isbn`]] of a book.

**Checked in this order:**

1. [[domain/catalog/operations/borrow-book/conditions/book-is-available]]: `book_unavailable`.
2. [[domain/catalog/operations/borrow-book/conditions/member-is-under-the-limit]]: `limit_reached`.

**Returns:** the `memberId` and the `isbn`.

**Satisfies:** [[purpose/goals/know-who-holds-each-book]] and [[purpose/stakeholders/staff/wishes/lend-a-book]].
```

A Condition states what must hold and nothing about its code, which the Operation gives.

`spec/domain/catalog/operations/borrow-book/conditions/book-is-available.md`

```markdown
---
kind: std/Condition
---
# Book is available

No member holds the [[domain/catalog/concepts/book]].
```

`spec/domain/catalog/operations/borrow-book/conditions/member-is-under-the-limit.md`

```markdown
---
kind: std/Condition
---
# Member is under the limit

The member holds fewer books than [[domain/members/concepts/member/invariants/within-the-loan-limit|the loan limit]] allows.
```

An Outcome states what holds afterwards; what no Outcome names does not change.

`spec/domain/catalog/operations/borrow-book/outcomes/member-holds-the-book.md`

```markdown
---
kind: std/Outcome
---
# Member holds the book

The member holds the [[domain/catalog/concepts/book]].
```

`markitect doc domain/catalog/operations/borrow-book` shows the Operation with its Conditions in that order and its Outcome, and the Rules that bind it without a link, such as the layout.

## Package interfaces/desk

A site is a Package under `interfaces/`, because its Pages reach the Operations of the areas, and Root may not link down into a child Package. An API or a command line is a Package of the same kind with an Interface, whose Routes name the method and path or the command, each with a `**Reaches:**` line that links its Operation.

`spec/interfaces/desk/desk.md`

```markdown
---
kind: Package
---
# Desk

The pages the staff use in a browser at the desk.

It keeps to itself how a page calls the service.
```

`spec/interfaces/desk/pages/borrow/borrow.md` is a Page with one Action as its part.

```markdown
---
kind: std/Page
---
# Borrow

**For:** `anyone`, at `/` in the browser at the desk.

**Shows:** what [[domain/catalog/operations/borrow-book]] returns for the last loan.
```

`spec/interfaces/desk/pages/borrow/actions/borrow.md`

```markdown
---
kind: std/Action
---
# Borrow

Asks for a `memberId` and an `isbn`, performs [[domain/catalog/operations/borrow-book]], and leads back to this page.
```

## Package interfaces/email

A message to a person, such as an e-mail, is a Message in `messages/` of a Package of its own under `interfaces/`. A call to another system would be a Connector instead.

`spec/interfaces/email/email.md`

```markdown
---
kind: Package
---
# E-mail

The e-mails the library sends to its members.

It keeps to itself how an e-mail is handed over.
```

`spec/interfaces/email/messages/loan-receipt.md` names the member's address by the Package that keeps it, since the address itself is internal to `domain/members`.

```markdown
---
kind: std/Message
---
# Loan receipt

Tells a member which book they borrowed.

**To:** the member who borrowed the book, at the address [[domain/members]] keeps for them.

**Sent when:** [[domain/catalog/operations/borrow-book/outcomes/member-holds-the-book]].

**Holds:** the `isbn` that [[domain/catalog/operations/borrow-book]] returns.
```

## A cycle, and its fix

Suppose the first draft of `member.md` also said `The books a member holds are listed in [[domain/catalog/concepts/book]].` `markitect check` refuses it:

```text
.markitect/spec/domain/catalog/operations/borrow-book/borrow-book.md:10: package.cycle: the Packages domain/catalog, domain/members depend on each other in a cycle: domain/catalog/operations/borrow-book/borrow-book links [[domain/members/concepts/member/attributes/member-id]]; domain/members/concepts/member/member links [[domain/catalog/concepts/book/book]]. Move the shared concept into a common parent, or invert one dependency
```

Moving the book or the member up to Root would not make sense here, so invert the dependency: `domain/catalog` is the side that needs members, so `domain/members` links nothing in `domain/catalog`. Delete that sentence, and `markitect list --deps` shows `domain/catalog` depending on `domain/members`, and the two interfaces on the areas.

## Registry pages and the ignore file

A row names a thing with parts by its folder, which covers its parts. The project Definition, the Goal, the Stakeholder, the Exclusion and the Term have no row: their Kinds declare `realized: false`, since no file realizes them. Root's page lists the files nobody else claims, including shared files, and names no patterns.

`registry.md`

```markdown
# Registry: Root

Root owns every file no other Package claims.

| File | Realizes | Why |
|---|---|---|
| `go.mod` | [[architecture/rules/layout]] | Go module of the whole service |
| `.github/workflows/ci.yml` | [[architecture/rules/layout]] | Runs `go test ./...` |
| `cmd/library/main.go` | [[architecture/rules/layout]] | Starts the server and wires the Packages |
```

Each Package claims its folder by pattern and lists every file in it. A test file lists what it checks, the Quality it measures included.

`registry/domain/members.md`

```markdown
---
paths:
  - internal/members/**
---
# Registry: domain/members

| File | Realizes | Why |
|---|---|---|
| `internal/members/member.go` | [[domain/members/members]] [[domain/members/concepts/member]] | The member and the loan limit |
| `internal/members/member_test.go` | [[domain/members/concepts/member]] | Tests |
```

`registry/domain/catalog.md`

```markdown
---
paths:
  - internal/catalog/**
---
# Registry: domain/catalog

| File | Realizes | Why |
|---|---|---|
| `internal/catalog/book.go` | [[domain/catalog/catalog]] [[domain/catalog/concepts/book]] | The book |
| `internal/catalog/borrow.go` | [[domain/catalog/operations/borrow-book]] | Borrowing |
| `internal/catalog/borrow_test.go` | [[domain/catalog/operations/borrow-book]] | Tests |
```

`registry/interfaces/desk.md`

```markdown
---
paths:
  - internal/desk/**
---
# Registry: interfaces/desk

| File | Realizes | Why |
|---|---|---|
| `internal/desk/pages.go` | [[interfaces/desk/desk]] [[interfaces/desk/pages/borrow]] | The page and its handler |
| `internal/desk/borrow_test.go` | [[interfaces/desk/pages/borrow]] [[quality/performance-efficiency]] | Tests, with the time of a loan |
```

`registry/interfaces/email.md`

```markdown
---
paths:
  - internal/email/**
---
# Registry: interfaces/email

| File | Realizes | Why |
|---|---|---|
| `internal/email/receipt.go` | [[interfaces/email/email]] [[interfaces/email/messages/loan-receipt]] | The receipt |
| `internal/email/receipt_test.go` | [[interfaces/email/messages/loan-receipt]] | Tests |
```

`ignore` lists `LICENSE`, which realizes nothing. What `markitect init` installs needs no line:

```text
# Legal text, realizes nothing
LICENSE
```

## Result

```text
$ markitect check
ok: 26 Definitions in 5 Packages, 12 files accounted
```

`markitect vet` shows no warnings. After the limit in `within-the-loan-limit.md` changes from `3` to `5`, `markitect status` lists `domain/catalog/operations/borrow-book/borrow-book` and `domain/members/concepts/member/member`, each as one item with its parts and with the files the registry names for it, such as `internal/catalog/borrow.go` and `internal/members/member.go`. The member is listed because its Invariant changed, the Operation because its Condition `member-is-under-the-limit` links the Invariant.

## Other errors you will meet

The project Definition gains the line `Staff lend books with [[domain/catalog/operations/borrow-book]].`:

```text
.markitect/spec/project.md:8: link.downward: Root links down to domain/catalog/operations/borrow-book/borrow-book inside its child Package domain/catalog; a parent may name a child Package but not a Definition inside it (move the shared concept up, or let the child link to a need stated here)
```

The project Definition may name the Package, `[[domain/catalog]]`, not a Definition inside it.

The Outcome `member-holds-the-book` gains the line `A receipt goes to the member's [[domain/members/concepts/member/attributes/email]].`:

```text
.markitect/spec/domain/catalog/operations/borrow-book/outcomes/member-holds-the-book.md:8: link.internal: domain/members/concepts/member/attributes/email is internal to domain/members; only domain/members and the Packages below it may link to it
```

Describe the need in `domain/catalog`'s own words, or make the Definition public if other Packages should build on it.

The page of `domain/catalog` gains the row ``| `internal/members/member.go` | [[domain/members/concepts/member]] | The member |``, and a file `newfile.txt` appears that no page lists:

```text
.markitect/registry/domain/catalog.md:12: registry.wrong_owner: internal/members/member.go is owned by domain/members, so it belongs on .markitect/registry/domain/members.md
newfile.txt: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root, or add it to the ignore file
```

List the file on its owner's page, or add it to the ignore file.
