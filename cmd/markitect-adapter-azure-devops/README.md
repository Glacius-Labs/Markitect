# Azure DevOps Git metadata adapter

This standalone, read-only command adapter compares the canonical
`data.defaultBranch` string of each explicitly mapped resource with a captured
Azure DevOps Git Repository response. It consumes only the normalized
`semantic-model/v1alpha1` request and staged capture files. It does not contact
Azure DevOps, request credentials, run another adapter, or mutate a repository.

The evidence contract is deliberately narrow: one captured `GET` response for
the repository metadata endpoint, API version `7.1`, and the response's
`id`, `project.id`, and `defaultBranch`. It does not establish that a capture
is authentic, current, authorized, or complete beyond those fields. A result
can show only that the supplied request/response record is internally
consistent with the declared org/project/repository mapping and canonical
branch value.

## Build

```powershell
go build -o .artifacts/adapters/markitect-adapter-azure-devops.exe ./cmd/markitect-adapter-azure-devops
```

## Capture and configuration

Each `captureFile` must be listed in the command adapter's exact `config.inputs`
so Markitect stages its fixed-snapshot bytes for the adapter. Preserve the
HTTP method, full request URL including `api-version`, status code, and JSON
response body in this envelope. Capture files must be regular files no larger
than 1 MiB. Duplicate JSON object keys are rejected to keep envelope and body
interpretation unambiguous; other Azure response fields remain allowed.

```json
{
  "request": {
    "method": "GET",
    "url": "https://dev.azure.com/example-org/11111111-1111-4111-8111-111111111111/_apis/git/repositories/22222222-2222-4222-8222-222222222222?api-version=7.1"
  },
  "response": {
    "status": 200,
    "body": {
      "id": "22222222-2222-4222-8222-222222222222",
      "project": { "id": "11111111-1111-4111-8111-111111111111" },
      "defaultBranch": "refs/heads/main"
    }
  }
}
```

Configure explicit stable target identity and per-resource repository mapping.
The resource's `data.defaultBranch` is the desired value; the adapter does not
duplicate it in provider parameters or infer mappings from names.

```yaml
spec:
  adapters:
    - name: azure-git-metadata
      type: command
      version: markitect-azure-devops/v0.1.0
      config:
        target: azure-devops://example-org/11111111-1111-4111-8111-111111111111
        inputs: [evidence/source-repository.json]
        observe: [markitect-adapter-azure-devops, observe]
        plan: [markitect-adapter-azure-devops, plan]
        verify: [markitect-adapter-azure-devops, verify]
        parameters:
          apiVersion: "7.1"
          organization: example-org
          projectId: 11111111-1111-4111-8111-111111111111
          repositories:
            - resource: engineering/software.markitect.org/v1alpha1/GitRepository/source
              repositoryId: 22222222-2222-4222-8222-222222222222
              captureFile: evidence/source-repository.json
```

The package accepts only `observe`, `plan`, and `verify`. A plan may report a
complete observation with error-severity `default-branch-drift` findings; its
`operations` list is always empty because evidence findings are not remote
actions. Verify fails when the captured observation differs from the saved
completed plan. Malformed or insufficient capture data, including a request
URL or response body for the wrong project/repository, is `incomplete`; a
declared adapter target inconsistent with its organization/project parameters
is `failed`.

The Microsoft endpoint and response shape are documented at [Get Repository,
REST API 7.1](https://learn.microsoft.com/en-us/rest/api/azure/devops/git/repositories/get-repository?view=azure-devops-rest-7.1).
