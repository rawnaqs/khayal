# CLI Color Rules — Strict

## The only 7 styles that exist

```go
theme.Bold            // section headers only
theme.Primary         // all content values
theme.Muted           // all keys and labels
theme.Dim             // timestamps, counts, hints, redacted
theme.SuccessStyle    // ✓ only
theme.ErrorStyle      // ✗ only
theme.ProcessingStyle // ⏳ only
```

## Mapping

```
section header  → theme.Bold
key             → theme.Muted
value           → theme.Primary
timestamp       → theme.Dim
count           → theme.Dim
score           → theme.Dim
hint            → theme.Dim
redacted        → theme.Dim
✓               → theme.SuccessStyle
✗               → theme.ErrorStyle
⏳              → theme.ProcessingStyle
tag badge       → theme.Tag
```

## Hard rules

1. Every printed string goes through theme. No exceptions.
2. No fmt.Println("raw string") ever.
3. No lipgloss.Color("#hex") outside theme package ever.
4. No err.Error() shown to user ever.
5. No fmt.Printf with mixed styled/unstyled strings ever.
6. Section labels always theme.Dim uppercase — TODAY, QUEUE, VAULT.
7. Tags always theme.Tag.Render() — never plain text.
8. Dividers always theme.Divider(width) — never literal ────.
9. Keys always width-padded to same column within a section.
10. One blank line between sections. Zero blank lines between keys.
