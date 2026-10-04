# Template-derived Go Todo API

Issue#1, Core#1666, SPEC-002 AC-4AJ.8. Development.

API-1: GitHub template_repository is service-lasso/service-template; input is generated develop and provenance records exact upstream develop. Go API owns source, manifest, package/test/verify and CI.
API-2: Managed todo-api depends on postgres, consumes resolved runtime allocation, opens PostgreSQL with bounded retries and exposes loopback allocated HTTP health/create/list. Input bounded, SQL parameterized, dependency outage explicit; existing todo IDs retained.
API-3: Produce Windows/Linux/macOS amd64 archive commands matching manifest. Packaging never includes credentials or data. Native runtime/package structure checks plus actual PostgreSQL integration and independent Windows Core-managed consumer proof remain distinct.
API-4: Release only explicitly dispatched development candidate after terminal CI and source review. No GA or platform acceptance inferred from cross-compilation.

Copied template producer/admission programme is upstream reference, not API acceptance. Protected upstream tests remain unchanged and no upstream publisher/admission claim is made. Echo-specific starter scripts are explicitly adapted to API-1–4, with separate review/direct evidence.
