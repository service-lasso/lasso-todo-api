# Template-derived Go Todo API

Issue#1, Core#1666, SPEC-002 AC-4AJ.8. Development.

API-1: GitHub template_repository is service-lasso/service-template; input is generated develop and provenance records exact upstream develop. Go API owns source, manifest, package/test/verify and CI.
API-2: Managed todo-api depends on postgres, consumes resolved runtime allocation, opens PostgreSQL with bounded retries and exposes loopback allocated HTTP health/create/list. Input bounded, SQL parameterized, dependency outage explicit; existing todo IDs retained.
API-3: Produce Windows/Linux/macOS amd64 archive commands matching manifest. Packaging never includes credentials or data. Native runtime/package structure checks plus actual PostgreSQL integration and independent Windows Core-managed consumer proof remain distinct.
API-4: Release only explicitly dispatched development candidate after terminal CI and source review. No GA or platform acceptance inferred from cross-compilation.

API-AUTH-1 (#5 / Core #1692): Resource-server authorization precedes every /todos handler and database operation. Runtime requires explicit anonymous mode for earlier local lessons or complete Zitadel mode; unset/unknown/partial configuration never silently permits access. Health is public.
API-AUTH-2: Validate access tokens through Zitadel HTTPS introspection using a registered API client and privately supplied secret file. Enforce active, exact issuer/project audience/issuing Todo client, user subject, Bearer token type, openid scope, expiration and not-before. Reject malformed/duplicate/oversized Authorization headers. Redirects, provider failures and malformed responses deny access; no positive cache masks revocation.
API-AUTH-3: Bound TLS requests and response sizes; local CA trust is explicit. Credential files, tokens and provider errors never appear in response/log/archive/manifest. Metadata declares the supported auth contract so paired Todo setup refuses older unsecured consumers.
API-AUTH-4: Tests directly cover successful protected GET/POST plus absent, wrong, revoked, expired, not-yet-valid, issuer/audience/client/type/scope rejection, configuration and provider outages before handler mutation; existing protected tests remain unchanged. Real Zitadel, released archive and managed PostgreSQL acceptance remain separate required gates.
API-AUTH-5: Coordinate Todo's server-side access-token forwarding and both stopped-service configuration; publish checksum-bound development consumers and corrected docs after independent review. Preserve earlier data and failures; no GA or production deployment.

Copied template producer/admission programme is upstream reference, not API acceptance. Protected upstream tests remain unchanged and no upstream publisher/admission claim is made. Echo-specific starter scripts are explicitly adapted to API-1–4, with separate review/direct evidence.

## API #7 import compatibility
- API-AUTH-6: Publish the non-secret TODO_API_AUTH_CONTRACT=zitadel-introspection-v1 in env, which Core manifest import preserves. Keep the same runtime enforcement, metadata and explicit mode. Literal acquired-manifest/helper checks precede released acceptance; prior stripped-metadata attempt is retained.
