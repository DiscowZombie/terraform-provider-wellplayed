# CLAUDE.md — WellPlayed project

This file tells Claude Code (and other AI agents) the rules for working in
this WellPlayed-powered project. Bundled skills under `.claude/skills/` go
deeper on individual SDKs and the `wellplayed` CLI; this file holds the
project-wide rules that apply *everywhere*.

If you change anything here, keep the **GraphQL Rule** section intact — it is
the single most important convention.

## Quick reference

- **GraphQL toolkit**: `@well-played.gg/typescript-sdk` exports a typed
  `graphql` function (powered by `gql.tada` + WellPlayed schema
  introspection). It statically types every operation against the live schema.
- **Clients** (both ship from `@well-played.gg/typescript-sdk`):
  - `createWellPlayedClient(...)` — fully-wired Apollo client (HTTP +
    WebSocket subscriptions + localStorage cache persist) for browser apps.
  - `createTypedClient(...)` — lower-level batched HTTP client suitable
    for Node scripts, CLI tools, and server-side code.
- **CLI**: `wellplayed` — see `.claude/skills/wellplayed-cli/SKILL.md`.
  Notable commands: `wellplayed graphql` for ad-hoc API calls,
  `wellplayed login`, `wellplayed install-skills` (also rewrites this
  file).
- **Schema lookups**: WellPlayed MCP at `https://mcp.well-played.gg/mcp`
  (already wired in `.mcp.json`).

## GraphQL Rule (MANDATORY)

**Always declare GraphQL documents with the typed `graphql` function exported
by `@well-played.gg/typescript-sdk`. Never use `gql` from `graphql`,
`graphql-tag`, or `@apollo/client`.**

Why this matters:

- `graphql` from `@well-played.gg/typescript-sdk` is built on
  `gql.tada` and the WellPlayed schema introspection. It infers the result
  type and variables type of every operation directly from the document.
- All client call sites (Apollo `query`/`mutate`/`subscribe`,
  `createTypedClient` calls, React hooks if you use the React SDK) infer
  their generics automatically — you never need to write `<TData, TVars>`
  or hand-author matching `interface`/`type` declarations.
- Schema mismatches (typos, removed fields, wrong argument types) are caught
  at compile time, not at runtime.
- `gql` from any other library returns an opaque `DocumentNode` and
  silently turns all of those guarantees off.

### Do this

```ts
import { graphql, type ResultOf } from '@well-played.gg/typescript-sdk';

const GET_MY_ACCOUNT = graphql(`
  query getMyAccount {
    getMyAccount {
      id
      email
      permissions { permission }
    }
  }
`);

type MyAccount = ResultOf<typeof GET_MY_ACCOUNT>['getMyAccount'];

// Apollo client (browser):
const { data } = await client.query({ query: GET_MY_ACCOUNT });
// data.getMyAccount is fully typed — no <TData> generic.
```

### Don't do this

```ts
// Forbidden — opaque DocumentNode, no schema check, no inferred types.
import { gql } from '@apollo/client';

const GET_MY_ACCOUNT = gql`
  query getMyAccount { getMyAccount { id email } }
`;

// Equally forbidden.
import { gql } from 'graphql-tag';
import gql from 'graphql-tag';
```

### Helper types

Need to extract a result, variables, or fragment type? Use the helpers
re-exported next to `graphql` — never re-derive the shape by hand.

```ts
import {
  graphql,
  type ResultOf,
  type VariablesOf,
  type FragmentOf,
  readFragment,
} from '@well-played.gg/typescript-sdk';

const Q = graphql(`query Foo { foo { id name } }`);

type Result = ResultOf<typeof Q>;     // { foo: { id: string; name: string } }
type Vars = VariablesOf<typeof Q>;    // operation variables, fully inferred
```

For fragments: declare them with `graphql(\`fragment ...\`)`, type them
with `FragmentOf<typeof MyFragment>`, and read them with `readFragment`.
Never spread the raw fragment payload into a fragment-typed slot — it
bypasses the masking gql.tada provides.

## Authentication

The typescript-sdk supports two flows:

- **User-token flow** — pass a `token` (OAuth/OIDC access token) you
  already have to `createWellPlayedClient` / `createTypedClient`. If your
  project also pulls in `@well-played.gg/react-sdk`, that flow is what its
  React provider wires up automatically.
- **Application flow** — pass `application: { clientId, clientSecret }`
  and the SDK exchanges them for a service token via the WellPlayed OAuth
  endpoint. Use this for backend services, CI jobs, and scripts.

Both flows require an `organizationId` (the org short id). In Node-side
code, never hardcode tokens or client secrets — load them from environment
variables.

## Working with the API

- **In code**: use `createWellPlayedClient` (browser, with subscriptions)
  or `createTypedClient` (Node, batched HTTP). Both accept typed
  `graphql(...)` documents directly — no per-call generics needed.
- **Ad-hoc / scripts / CI**: use `wellplayed graphql`. It accepts an
  inline query, a `-f` file, or stdin, and supports typed `-F`
  variables. Full reference: `.claude/skills/wellplayed-cli/SKILL.md`.
- **Schema exploration**: prefer the WellPlayed MCP server over manually
  crawling the introspection blob. Claude Code is already wired through
  `.mcp.json`.

## TypeScript conventions

- **Strict mode stays on.** Don't loosen `tsconfig.json`.
- **No `any` / `unknown` shortcuts.** The typed `graphql(...)` document
  is the escape hatch for API types — there is no excuse to fall back to
  `any` for query results.
- **No `as` casts** to massage a query result into a different shape.
  Re-shape the document instead.
- **No `// @ts-ignore` / `// @ts-expect-error`** to silence a schema
  mismatch. Either the document is wrong, the schema changed (regenerate
  introspection), or the field really doesn't exist — never silence it.
- **Type every function signature** explicitly. `gql.tada`'s inferred
  generics already flow through the SDK clients; lean on that, don't
  bypass it.

## What lives where

| Path | What |
|------|------|
| `.claude/skills/` | Bundled SDK + CLI skills (regenerate with `wellplayed install-skills`) |
| `.mcp.json` | MCP server entries; keeps Claude Code wired to the WellPlayed schema |
