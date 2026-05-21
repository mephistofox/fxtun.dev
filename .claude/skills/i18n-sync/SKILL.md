---
name: i18n-sync
description: Add, rename, or audit vue-i18n translation keys across fxTunnel frontends, keeping ru/en locales in parity. Use when adding user-facing strings to web/ or the GUI, or when checking for missing translations.
---

# i18n-sync

fxTunnel has two vue-i18n frontends, each with paired `en.json` + `ru.json`:

- `web/src/i18n/{en,ru}.json` — marketing site + user dashboard (~1200 keys)
- `gui/frontend/src/i18n/{en,ru}.json` — Wails desktop GUI (~330 keys)

(`admin/` has no i18n — single language, skip it.)

Every key MUST exist in both locales of a frontend, or vue-i18n falls back / warns and the UI shows raw keys. The two domains are localized separately per the domain strategy: **fxtun.ru = Russian, fxtun.dev = English** — so both translations matter, neither is a placeholder.

## Audit parity

Run the bundled checker from the repo root:

```bash
.claude/skills/i18n-sync/check-parity.sh
```

It lists any key present in one locale but missing in its pair and exits non-zero on mismatch. Run it before and after edits.

## Add a new key

1. Decide which frontend(s) need it (`web` and/or `gui/frontend`). A string shown in both surfaces must be added to both.
2. Add the key to **both** `en.json` and `ru.json` of each target frontend, at the **same nested path**, preserving the existing structure (top-level groups like `auth`, `dashboard`, `common`, `checkout`, …).
3. Match interpolation syntax already in use: named `{name}` and literal `{'|'}` (a CSP-safe custom compiler in `index.ts` handles these — do NOT introduce `@:linked` or `new Function`-style messages).
4. Keep JSON valid and key ordering consistent with neighbors. Provide a real translation for each locale (en = English, ru = Russian), not a copy of the other language.
5. Re-run the parity checker; then `cd <frontend> && npx vue-tsc` (the build uses `vue-tsc && vite-ssg build`) to confirm types still resolve.

## Rename / remove a key

Apply the identical change to all four files in lockstep (or two, if the key lives in only one frontend), then grep the frontend `src/` for `t('old.key')` / `$t("old.key")` usages and update them. Re-run the checker.

## Notes

- Don't edit `index.ts` for ordinary key work — it only holds the i18n setup and the custom message compiler.
- Commit scope: `feat(web)` / `feat(gui)` / `chore(i18n)` per Conventional Commits.
