# Example: a lending library

A small Go service that lends books to members over REST. It compiles as written, and the outputs shown are what the tools print. To stay short it leaves out returning a book. The spec has Root and two Packages, `members` and `catalog`.

## The repository

```text
README.md  LICENSE  go.mod
.github/workflows/ci.yml
cmd/library/main.go
internal/members/{member.go, http.go, member_test.go}
internal/catalog/{book.go, loans.go, loans_test.go}
.markitect/
  manifest.yaml  lock.yaml  ignore  registry.md
  registry/{members.md, catalog.md}
  spec/
    project.md  layout.md  error-body.md
    members/{members, member, member-id, email, loan-limit, register-member, post-members}.md
    catalog/{catalog, book, borrow-book, post-loans}.md
```

`manifest.yaml` names the Libraries; run `markitect tidy` after changing it:

```yaml
libraries:
  std: 1.1.0
  rest: 1.1.0
```

## Root: anchor and Rules

Root holds what binds the whole project. The project's anchor names the Packages by link and says what is out of scope.

`spec/project.md`

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

`spec/error-body.md` is a Rule both Packages' Endpoints follow, so it sits at their common parent.

```markdown
---
kind: std/Rule
---
# Error body

Every refusal of the REST interface answers with a 4xx status and the JSON body `{"error": "<code>"}`, where `<code>` is the lower-case code named by the Endpoint, such as `limit_reached`. The body carries nothing else.
```

`spec/layout.md` is a layout convention, so a Rule at Root.

```markdown
---
kind: std/Rule
---
# Layout

Go code lives under `internal/<package>/` with its tests in the same folder as `*_test.go`. The program's entry point is `cmd/library/main.go`. Every Package of the spec owns the folder `internal/<package>/`.

Tests run with `go test ./...`.
```

The project needs no Kind of its own. A project Kind earns its place when several Definitions share a form that no Library Kind gives; one limit is a Rule.

## Package members

The anchor, Concepts, the member's identity, a value rule that is internal, a Rule only this Package follows, a Feature and an Endpoint. Links are full paths even inside the Package.

`spec/members/members.md` is the Package's anchor: a Definition of kind `Package` that carries the folder's name makes the folder a Package, and says what it is responsible for and what it keeps to itself. A folder without one, such as `spec/members/endpoints/`, would be a namespace of `members` and nothing more.

```markdown
---
kind: Package
---
# Members

Who is registered with the library, and how much one person may borrow at once.

It keeps to itself how an address is compared with another.
```

`spec/members/member.md` (after the cycle below is fixed)

```markdown
---
kind: std/Concept
---
# Member

A person registered with the library.

- A member has a `name` and an [[members/email|email]], which no other member has.
- A member is `active` or `suspended`. A suspended member keeps their data.
- A member is identified by its [[members/member-id|member identifier]].
```

`spec/members/member-id.md` is the member's identity, a Definition of its own: other Packages name a member by it without relying on the rest of what a member is. The member and its identity link each other, which the spec allows inside one Package.

```markdown
---
kind: std/Concept
---
# Member identifier

The identifier of a [[members/member]]: an opaque string the library issues at registration, which never changes and is never reused.
```

`spec/members/loan-limit.md` is a Rule of this Package. It states the limit and nothing about what happens when it is reached: the Feature that refuses says that, once.

```markdown
---
kind: std/Rule
---
# Loan limit

A [[members/member]] may hold at most `3` books at once. The limit applies to every member alike.
```

`spec/members/email.md` is a value rule of the project's own, stated once, in the only Package that uses it. It is `internal: true`: registering builds on it, other Packages must not.

```markdown
---
kind: std/Concept
internal: true
---
# Email

An email address. It is valid when it holds an `@`, and two emails are the same when they differ only in upper and lower case.
```

`spec/members/register-member.md` is a Feature.

```markdown
---
kind: std/Feature
---
# Register member

A visitor becomes a [[members/member]] by giving a name and an email.

- The new member is `active`.
- An [[members/email|email]] that another member already has is refused with `email_taken`.
- A missing name or an email that is not valid is refused with `invalid_member`.
```

`spec/members/post-members.md` exposes it as a REST Endpoint and links the Feature and the Root Rule.

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

`spec/catalog/catalog.md` is its anchor.

```markdown
---
kind: Package
---
# Catalog

Which titles the library owns, and who holds each copy.

It keeps to itself how a loan is stored.
```

`spec/catalog/book.md`

```markdown
---
kind: std/Concept
---
# Book

A title the library owns, with one physical copy.

- A book has a `title`, an `author` and an `isbn`, which is unique.
- A book is `available` or `borrowed`. A borrowed book is held by exactly one [[members/member-id|member]].
```

The book names its holder by the member's identity: a new field of the member does not concern the book. Borrowing relies on the member's status, so it links the member itself.

`spec/catalog/borrow-book.md` is the Feature.

```markdown
---
kind: std/Feature
---
# Borrow book

An `active` [[members/member]] takes an `available` [[catalog/book]] home.

- The book becomes `borrowed` and is held by the member.
- A suspended member is refused with `member_suspended`.
- A book that is already `borrowed` is refused with `book_unavailable`.
- A member or a book the library does not know is refused with `not_found`.
- A member who has reached the [[members/loan-limit]] is refused with `limit_reached`.
```

`spec/catalog/post-loans.md` is its Endpoint.

```markdown
---
kind: rest/Endpoint
---
# POST /loans

Exposes [[catalog/borrow-book]] over HTTP.

- Request: the JSON body `{"member": "<member id>", "isbn": "<isbn>"}`, where `member` is a [[members/member-id|member identifier]].
- `201` with the body `{"member": "...", "isbn": "..."}`.
- `403` with `member_suspended` or `limit_reached`, `404` with `not_found` and `409` with `book_unavailable`, as in [[catalog/borrow-book]], in the body described by [[error-body]].
```

## A cycle, and its fix

Suppose the first draft of `members/member.md` also said, as its last bullet, `The books a member holds are listed in [[catalog/book]].` Now `catalog/book` links `members/member-id` and `members/member` links `catalog/book`. `markitect check` refuses it:

```text
.markitect/spec/catalog/book.md:9: package.cycle: the Packages catalog, members depend on each other in a cycle: catalog/book links [[members/member-id]]; members/member links [[catalog/book]]. Move the shared concept into a common parent, or invert one dependency
```

Moving `Book` or `Member` up to Root would not make sense here, so invert the dependency: `catalog` is the side that needs members, so `members` links nothing in `catalog`. Delete that bullet. `catalog/book` keeps its link, and `markitect list --deps` shows one dependency, `catalog` on `members`.

## Registry pages and the ignore file

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
| `internal/members/member.go` | [[members/members]] [[members/member]] [[members/member-id]] [[members/email]] [[members/register-member]] [[members/loan-limit]] | Member type, registration and the loan limit |
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
| `internal/catalog/book.go` | [[catalog/catalog]] [[catalog/book]] | Book type |
| `internal/catalog/loans.go` | [[catalog/borrow-book]] [[catalog/post-loans]] [[error-body]] | Borrowing and the `POST /loans` handler |
| `internal/catalog/loans_test.go` | [[catalog/borrow-book]] [[members/loan-limit]] | Tests |
```

`ignore` lists `LICENSE`, which realizes nothing. What `markitect init` installs needs no line:

```text
# Legal text, realizes nothing
LICENSE
```

## Result

```text
$ markitect check
ok: 14 Definitions in 3 Packages, 10 files accounted
```

`markitect vet` shows no warnings. After the limit in `loan-limit.md` changes from `3` to `5`, `markitect status` lists `members/loan-limit` and `catalog/borrow-book`, the one Definition that links to it, each with the files the registry names for it, such as `internal/members/member.go` and `internal/catalog/loans.go`. The member does not link back to the limit, so it stays off the list.

## Other errors you will meet

The project's anchor gains the line `Staff lend books with [[catalog/post-loans]].`:

```text
.markitect/spec/project.md:15: link.downward: Root links down to catalog/post-loans inside its child Package catalog; a parent may name a child Package but not a Definition inside it (move the shared concept up, or let the child link to a need stated here)
```

The project's anchor may name the Package, `[[catalog]]`, not a Definition inside it.

`catalog/borrow-book` gains the bullet `- The receipt goes to the member's [[members/email]].`:

```text
.markitect/spec/catalog/borrow-book.md:13: link.internal: members/email is internal to members; only members and the Packages below it may link to it
```

Describe the need in `catalog`'s own words, or make the Definition public if other Packages should build on it.

The page of `catalog` gains the row ``| `internal/members/http.go` | [[members/post-members]] | The handler |``, and a file `newfile.txt` appears that no page lists:

```text
.markitect/registry/catalog.md:12: registry.wrong_owner: internal/members/http.go is owned by members, so it belongs on .markitect/registry/members.md
newfile.txt: file.unregistered: neither registered nor ignored: list it on .markitect/registry.md, the page of its owner Root, or add it to the ignore file
```

List the file on its owner's page, or add it to the ignore file.
