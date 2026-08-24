# Ingest chunks raw capture content; reindex chunks the full note file

When a note is captured, chunk embeddings are built from the raw submitted
content only (no title, summary, or key ideas). When `khayal reindex` rebuilds
chunks from the vault, it chunks the full note body after stripping frontmatter —
including the generated `Summary` and `Key Ideas` sections. The two paths
therefore index slightly different text for the same note, and search coverage
differs depending on how a note's vectors were last written.

We accepted this asymmetry deliberately: capture must stay fast (chunking only
the user's text keeps the ingest path minimal), while reindex is a bulk repair
tool whose job is maximum coverage of what exists on disk. Unifying them would
require the capture path to re-read and re-parse its own written note, or reindex
to reconstruct raw-only text by locating the `## Raw` section — added coupling
and fragility in both directions, for no user-visible benefit we could identify.

## Consequences

- A note indexed at capture time and the same note after `khayal reindex` will
  not necessarily rank identically in semantic search.
- Reindexed notes can match queries against summary/key-idea phrasing that
  freshly-captured notes cannot, until they too are reindexed.
- If this becomes a real quality problem, revisit via one of: a `## Raw`
  marker convention that reindex can rely on, storing raw-only body length in
  frontmatter, or having capture hand reindex the same text it embeds.
