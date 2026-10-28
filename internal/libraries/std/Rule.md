---
kind: Kind
---
# Rule

A Rule Definition states something every artifact in its scope must satisfy: a policy, a constraint, a convention or a quality requirement.

- Its scope is its location. A Rule in the Root Package binds the whole project; a Rule in a Package binds that Package and every Package below it.
- State it so that a reviewer can decide whether an artifact conforms: what must hold, and where it applies when that is narrower than the scope.
- Give exact values in backticks: limits, error codes, names.
- Say why when the reason helps a reader apply the Rule to a case it does not name.

Layout conventions, such as where tests live, are Rules in the Root Package.
