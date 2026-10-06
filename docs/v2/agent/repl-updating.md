# Updating kdeps

`/upgrade` checks GitHub and, for a standalone install, downloads and verifies the new binary. The other REPL behaviors stay on [REPL features](/agent/repl).

kdeps checks GitHub for a newer stable release at startup (throttled to once every 24 hours, cached at `~/.kdeps/update-check.json`, bounded to 3 seconds) and, if one exists, prints a one-line notice under the banner:

```
Update available: v2.8.0 -> v2.9.0. Run /upgrade to update.
```

Run `/upgrade` (or `kdeps --upgrade` outside the REPL) any time to check immediately and install. What happens next depends on how kdeps was installed:

- **Homebrew** (`brew install kdeps/tap/kdeps`): prints `brew upgrade kdeps` instead of touching the binary - self-replacing it would desync Homebrew's own bookkeeping.
- **.deb / .apk package**: prints the matching package-manager upgrade command.
- **Standalone** (the `curl | sh` installer, or a manually downloaded binary): after a `[Y/n]` confirmation (skippable with `KDEPS_YES=1`), downloads the release archive for your platform, verifies its SHA256 against `checksums.txt`, and atomically replaces the running binary. Restart kdeps afterward.

## Nightly builds

kdeps also cuts a nightly build from `main` most days. `/upgrade nightly` (or `kdeps --upgrade --nightly`) switches the channel for that one check: it installs the latest nightly instead of the latest stable.

Nightly opt-in only works for a standalone install - Homebrew/.deb/.apk only ever track stable, so on those `/upgrade nightly` prints standalone-install instructions instead of a package-manager command. "Already up to date" for the nightly channel means you are running that exact nightly tag: a nightly reuses the current stable version number until the next stable release ships, so it is always offered until you are actually on it.

## Installing a specific version (including a downgrade)

`/upgrade <version>` (or `kdeps --upgrade --target-version <version>` outside the REPL) installs exactly that version, skipping the "is an update available" check entirely - the version was named explicitly, so kdeps installs it whether it is newer than the running build (upgrade), older (downgrade), or the same (reinstall):

```
/upgrade 2.35.0
Downgrade to v2.35.0 now? [Y/n]
```

Same standalone-only restriction as nightly: Homebrew/.deb/.apk cannot be told to install a specific (especially older) version, so those print a link to the release page instead.
