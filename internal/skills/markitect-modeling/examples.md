# Example: a lending library

A small Go service that lends books to members over REST. This example compiles with `markitect check`. The model has Root and two Packages, `members` and `catalog`.

## The repository

```text
README.md  LICENSE  go.mod
.github/workflows/ci.yml
cmd/library/main.go
internal/members/{member.go, http.go, member_test.go}
internal/catalog/{book.go, loans.go, loans_test.go}
.markitect/
  manifest.yaml  lock.yaml  execution.yaml  ignore.md  registry.md
  registry/{members.md, catalog.md}
  model/
    project.md  layout.md  error-body.md  Policy.md
    members/{member, loan-limit, open-loan-count, register-member, post-members}.md
    catalog/{book, borrow-book, post-loans}.md
```

`manifest.yaml` names the Libraries; run `markitect lock` after changing it:

```yaml
libraries:
  std: 1.0.0
  rest: 1.0.0
```

## Root: anchor, Rules, a project Kind

Root holds what binds the whole project. The anchor names the Packages by link and says what is out of scope.

`model/project.md`

```markdown
---
kind: std/Project
---
# Lending Library

A small service that lends books to registered members, for a neighborhood library whose staff want to see who holds which book.

Staff and members use it through a REST interface. It is written in Go.

- [[members]] registers members and decides how many books a member may hold.
- [[catalog]] holds the books and lends them to members.

Out of scope: payments, reminders and reservations.
```

`model/layout.md` is a layout convention, so a Rule at Root. `model/error-body.md` is a Rule both Packages' Endpoints follow, so it sits at their common parent.

```markdown
---
kind: std/Rule
---
# Error body

Every refusal of the REST interface answers with a 4xx status and the JSON body `{"error": "<code>"}`, where `<code>` is the lower-case code named by the Endpoint, such as `limit_reached`. The body carries nothing else.
```

```markdown
---
kind: std/Rule
---
# Layout

Go code lives under `internal/<package>/` with its tests in the same folder as `*_test.go`. The program's entry point is `cmd/library/main.go`. Every Package of the model owns the folder `internal/<package>/`.

Tests run with `go test ./...`.
```

No Library Kind fits a tunable limit, so the project defines one. `model/Policy.md` is a Kind (`kind: Kind`); it sits at Root because `members` uses it today and others may later.

```markdown
---
kind: Kind
---
# Policy

A Policy Definition states one tunable limit of the lending service.

- Name the limit and give its value in backticks.
- Say who it applies to and what a caller observes when it is reached: the refusal code in backticks.
- Say why the value was chosen only when the reason helps a reader apply it to a case it does not name.
- Leave how the limit is enforced to the realization.
```

## Package members

Concepts, a Feature, an Endpoint, the project Kind in use and one internal Definition. Links are full paths even inside the Package.

`model/members/member.md` (after the cycle below is fixed)

```markdown
---
kind: std/Concept
---
# Member

A person registered with the library. Only a member can borrow books.

- A member has a `name` and an `email`, which is unique among members.
- A member is `active` or `suspended`. A suspended member keeps their data but cannot borrow.
- A member holds at most as many books at once as [[members/loan-limit]] allows.
```

`model/members/loan-limit.md` uses the project Kind. `model/members/open-loan-count.md` is `internal: true`: the loan limit builds on it, other Packages must not.

```markdown
---
kind: Policy
---
# Loan limit

A [[members/member]] may hold at most `3` books at once, counted as in [[members/open-loan-count]]. A fourth loan is refused with `limit_reached`.

The limit applies to every member alike.
```

```markdown
---
kind: std/Concept
internal: true
---
# Open loan count

The number of books a [[members/member]] has borrowed and not yet returned. A book that is overdue still counts.

This is how the realization reads the count; other Packages ask the [[members/loan-limit]], not the count.
```

`model/members/register-member.md` is a Feature; `model/members/post-members.md` exposes it as a REST Endpoint and links the Feature and the Root Rule.

```markdown
---
kind: std/Feature
---
# Register member

A visitor becomes a [[members/member]] by giving a name and an email.

- The new member is `active`.
- An email that another member already has is refused with `email_taken`.
- A missing name or an email without an `@` is refused with `invalid_member`.
```

```markdown
---
kind: rest/Endpoint
---
# POST /members

Exposes [[members/register-member]] over HTTP.

- Request: the JSON body `{"name": "...", "email": "..."}`.
- `201`: the new member as `{"id": "...", "name": "...", "email": "...", "status": "active"}`.
- `409` with `email_taken` and `422` with `invalid_member`, as in [[members/register-member]], in the body described by [[error-body]].
```

## Package catalog

`catalog` needs members (to borrow, a member must be active and under the limit), so it depends on `members`; `members` must never depend back.

`model/catalog/book.md`

```markdown
---
kind: std/Concept
---
# Book

A title the library owns, with one physical copy.

- A book has a `title`, an `author` and an `isbn`, which is unique.
- A book is `available` or `borrowed`. A borrowed book is held by exactly one [[members/member]].
```

`model/catalog/borrow-book.md` (the Feature) and `model/catalog/post-loans.md` (its Endpoint):

```markdown
---
kind: std/Feature
---
# Borrow book

An `active` [[members/member]] takes an `available` [[catalog/book]] home.

- The book becomes `borrowed` and is held by the member.
- A suspended member is refused with `member_suspended`.
- A book that is already `borrowed` is refused with `book_unavailable`.
- A member who has reached the [[members/loan-limit]] is refused with `limit_reached`.
```

```markdown
---
kind: rest/Endpoint
---
# POST /loans

Exposes [[catalog/borrow-book]] over HTTP.

- Request: the JSON body `{"member": "<member id>", "isbn": "<isbn>"}`.
- `201`: the loan as `{"member": "...", "isbn": "...", "dueDate": "YYYY-MM-DD"}`.
- `403` with `member_suspended` or `limit_reached`, and `409` with `book_unavailable`, as in [[catalog/borrow-book]], in the body described by [[error-body]].
```

## A cycle, and its fix

Suppose the first draft of `members/member.md` also said, as its last bullet, `The books a member holds are listed in [[catalog/book]].` Now `catalog/book` links `members/member` and `members/member` links `catalog/book`. `markitect check` refuses it:

```text
.markitect/model/catalog/book.md:9: package.cycle: the Packages catalog, members depend on each other in a cycle: catalog/book links [[members/member]]; members/member links [[catalog/book]]. Move the shared concept into a common parent, or invert one dependency
```

Moving `Book` or `Member` up to Root would not make sense here, so invert the dependency: `catalog` is the side that needs members, so `members` states nothing about books. Delete that bullet. `catalog/book` keeps its link, and `markitect graph` shows one arrow, `catalog --> members`.

## Registry pages and the ignore page

Root's page lists the files nobody else claims, including shared files, and links every Definition each realizes. Root names no patterns.

`registry.md`

```markdown
# Registry: Root

Root owns every file no other Package claims.

| File | Realizes | Why |
|---|---|---|
| `README.md` | [[project]] | Project overview for readers |
| `go.mod` | [[project]] [[layout]] | Go module of the whole service |
| `.github/workflows/ci.yml` | [[layout]] | Runs `go test ./...` |
| `cmd/library/main.go` | [[project]] [[members/post-members]] [[catalog/post-loans]] [[error-body]] | Starts the REST server and wires both Packages |
```

Each Package claims its folder by pattern and lists every file in it. A test file lists the Definitions it checks.

`registry/members.md`

```markdown
---
paths:
  - internal/members/**
---
# Registry: members

| File | Realizes | Why |
|---|---|---|
| `internal/members/member.go` | [[members/member]] [[members/register-member]] [[members/loan-limit]] [[members/open-loan-count]] | Member type, registration and the loan limit |
| `internal/members/http.go` | [[members/post-members]] [[error-body]] | The `POST /members` handler |
| `internal/members/member_test.go` | [[members/member]] [[members/register-member]] [[members/loan-limit]] | Tests |
```

`registry/catalog.md`

```markdown
---
paths:
  - internal/catalog/**
---
# Registry: catalog

| File | Realizes | Why |
|---|---|---|
| `internal/catalog/book.go` | [[catalog/book]] | Book type |
| `internal/catalog/loans.go` | [[catalog/borrow-book]] [[catalog/post-loans]] [[error-body]] | Borrowing and the `POST /loans` handler |
| `internal/catalog/loans_test.go` | [[catalog/borrow-book]] [[members/loan-limit]] | Tests |
```

`ignore.md` keeps the lines `markitect init` wrote and adds `LICENSE`, which realizes nothing:

```markdown
# Ignored files

Files that deliberately belong to no Package.

| Path | Why |
|---|---|
| `.claude/skills/markitect-*/**` | Skills installed by markitect init |
| `.agents/skills/markitect-*/**` | Skills installed by markitect init |
| `LICENSE` | Legal text, realizes nothing |
```

## Result

```text
$ markitect check
ok: 12 Definitions in 3 Packages, 10 files accounted
```

`markitect lint` shows no warnings. `markitect impact --base HEAD` after changing the limit in `loan-limit.md` from `3` to `5` reaches `catalog/borrow-book`, `members/member` and `members/open-loan-count` through their links to it, and `internal/members/member.go` and `internal/catalog/loans.go` through the registry. Two Managers, `members` and `catalog`, work; Root reviews.

## Other errors you will meet

```text
project.md:14: link.downward: Root links down to catalog/post-loans inside its child Package catalog ...
```

The anchor may name the Package, `[[catalog]]`, not a Definition inside it.

```text
catalog/post-loans.md:11: link.internal: members/open-loan-count is internal to members; only members and the Packages below it may link to it
```

Describe the need in `catalog`'s own words, or make the Definition public if other Packages should build on it.

```text
registry/catalog.md:12: registry.wrong_owner: internal/members/http.go is owned by members, so it belongs on .markitect/registry/members.md
newfile.txt: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root, or add it to the ignore page
```

List the file on its owner's page, or add it to the ignore page.
