# Clarify connector

Read-only deep connector for the Clarify CRM REST API (JSON:API).

## Auth and requests

- API key sent as `Authorization: api-key <key>` (not Bearer). A workspace key sees only the
  workspace's public data; a personal key sees everything its creator can, including private
  meeting notes and summaries ([Authentication](https://developer.clarify.ai/docs/getting-started/authentication)).
- Base URL `https://api.clarify.ai`. Every path is `/v1/workspaces/{workspace}/...`; the slug is
  the connection's `workspace` metadata, added in code. Proxy callers put it in their own path.

## Objects

| Object | Endpoint | Metadata |
|---|---|---|
| `person`, `company`, `deal`, `meeting`, `task` | `GET .../objects/{object}/resources` | live, `GET .../schemas` |
| `c_*` custom objects | same | live, `GET .../schemas` |
| `users` | `GET .../users` | catalog generated from the OpenAPI spec |

Comments have no list endpoint (only each record's activity stream), and transcripts and recording
artifacts sit under per-meeting paths, so neither is an object here.

## Record shape

Read promotes each resource's `attributes` to top-level fields, named as `/schemas` names them.
The resource `type` (the object name) is dropped, as meetings have a `type` attribute of their own.
Many-to-one relationships are attributes (`company_id`). One-to-many and many-to-many relationships
(a deal's `people`, a meeting's `users`, a person's `deals`) are not: they are asked for with
`include`, and Read returns each as a list of ids. Only `/schemas` says which requested fields are
such relationships, so the first page of every record-object read needs `/schemas` (about 240 KB
in the test workspace), even when no `include` results; the connector reads it once per instance and
shares it with `ListObjectMetadata`. If `/schemas` fails, the read fails. `_created_by` and
`_updated_by` are opaque actor ids (`system`, or a user's auth-provider id), so they are strings,
not references to `users`. `Raw` is the resource untouched.

## Pagination

`page[limit]` (max 500) and `page[offset]`. `links.next` is null on the last page, but the link
itself comes back as `http://api.clarify.ai/...`, which does not answer, so the next page is the
request with `page[offset]` set to `meta.offset + meta.limit`.

## Incremental read

Record objects filter on `_updated_at`. The filter ignores the milliseconds of the value it is given
(a record came back for `Is after` its own timestamp), so the provider gets the window widened to
whole seconds (`Is on or after` Since floored, `Is before` Until's next second) and the connector
trims it to `(Since, Until]`. `/users` takes no filter; users are read in full and trimmed the same
way. With `ReadParams.Deleted`, record objects read `.../deleted-resources` (deletions from the last
30 days), filtered on `_deleted_at` in epoch seconds.

## Rate limits

3,000 requests per minute per workspace per endpoint, rolling; 429 carries `Retry-After`
([Rate limits](https://developer.clarify.ai/docs/api-basics/rate-limits)).
