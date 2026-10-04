# Lasso Todo API

GitHub-generated from `service-lasso/service-template`; exact develop provenance is recorded in `template-origin.json`. This separately managed Go/PostgreSQL `todo-api` service is the third progressive tutorial addition.

Use Node22+ and Go1.22+, then `npm ci`, `npm test`, `npm run package`, `npm run verify`. The template package/test/verify entrypoints are adapted for this service. Platform archives contain the native runtime at the command declared in service.json, never retained database data or credentials.

Verification alone checks native archive structure. Set `TODO_VERIFY_DATABASE_STATE` to an actual owned PostgreSQL allocation file for real SQL/create/list/restart verification. CI's required PostgreSQL job tests the held Linux archive produced by the package job. Windows Core-managed acceptance is separate; compilation is not macOS runtime qualification.

The manifest depends on managed `postgres`, uses its actual allocation, binds loopback HTTP and checks readiness against real SQL. Import an explicit published candidate through Core, then Install/Configure/Start in the same inventory. Configure the separate Todo UI service for its API stage. Public SQL defaults are for isolated local learning.

Candidate publication is explicitly dispatched from develop after package and held-archive SQL gates pass. Assets include exact source/run metadata, manifest and checksums. No GA declaration or inherited template admission/publisher programme qualification is implied.

## Inherited template links

The following upstream links describe the starting template; the active API scope is SPEC-TODO-API / issue#1.

Turn an existing program into a service that Lasso can install, configure, start, check, and package.

**[Create your service from this template](https://github.com/service-lasso/service-lasso/blob/develop/docs/components/service-template/bootstrap-new-service-repo.md)**

Use GitHub's **Use this template** button, rename the sample, replace its runtime payload, and describe it in `service.json`.

Validate your first package:

```powershell
pwsh -NoLogo -NoProfile -File ./scripts/package.ps1
pwsh -NoLogo -NoProfile -File ./scripts/test.ps1
```

[Write the manifest](https://github.com/service-lasso/service-lasso/blob/develop/docs/components/service-template/service-json-reference.md) · [Package it](https://github.com/service-lasso/service-lasso/blob/develop/docs/components/service-template/packaging.md) · [Validate it](https://github.com/service-lasso/service-lasso/blob/develop/docs/components/service-template/validation.md)

Want an application with ready-made dependencies? Start with [PostgreSQL and a small app](https://github.com/service-lasso/service-lasso/blob/develop/docs/first-useful-service.md) or an [app template](https://github.com/service-lasso/service-lasso/blob/develop/docs/reference-apps.md).

Reader guides live in Service Lasso. [Maintainer context](docs/maintainer-context.md) and implementation specs stay with the code.
