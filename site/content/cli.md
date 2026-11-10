# CLI

`asyncgo` has four commands. `generate` writes the document, `check` fails when the
committed document no longer matches your code, and `version` / `completion` /
`help` come from Cobra.

| Command | What it does |
| --- | --- |
| `asyncgo generate [dir]` | Write `asyncapi.yaml` or `asyncapi.json` for the module rooted at `dir` |
| `asyncgo check [dir]` | Fail if the generated document is out of date |
| `asyncgo version` | Print the asyncgo version |
| `asyncgo completion <shell>` | Print a shell completion script |
| `asyncgo help` | Built-in help for any command |

`--help` works on every command, and `--version` on the root.

!!! note "Invoking the CLI"

    The examples on this page call the bare `asyncgo` binary installed with
    `go install`. If you pinned the CLI as a module tool, prefix every command
    with `go tool` — `go tool asyncgo generate .` — and apply the same prefix to
    the transcripts below. Flags, exit codes and output are identical.
    [Getting started](getting-started.md#alternative-pin-the-cli-as-a-module-tool)
    has the setup.

## `asyncgo generate`

```console
$ asyncgo generate [dir] [flags]
```

| Flag | Effect |
| --- | --- |
| `[dir]` | Module root to load. Defaults to `.` |
| `--format` `yaml`\|`json` | Encoding of the output. Defaults to `yaml` |
| `-o`, `--output` | Output file, or an existing directory (trailing slash) to write `asyncapi.<format>` into. Defaults to `<dir>/asyncapi.<format>` |
| `--config` | Schema-mapping file. Defaults to `<dir>/asyncgo.yaml` |

```bash
# write asyncapi.yaml (committed artifact)
asyncgo generate .

# write asyncapi.json instead (2-space indented, newline-terminated)
asyncgo generate . --format json

# write to a custom location (a file path, or a directory with a trailing slash)
asyncgo generate . -o ./docs/asyncapi.yaml
asyncgo generate . -o ./docs/ --format json    # -> ./docs/asyncapi.json

# an explicit -o path wins whatever its extension: --format picks the encoding
asyncgo generate . -o ./docs/spec.yaml --format json

# read schema mappings from a file other than <dir>/asyncgo.yaml
asyncgo generate . --config ./ci/mappings.yaml
```

`-o` names the destination, `--format` picks the encoding: `-o spec.yaml --format json`
writes JSON to `spec.yaml`. A directory passed to `-o` must already exist.

## `asyncgo check`

`check` compares the document **byte for byte** against the file `generate` would
write for the same `--format`, `--output` and `--config` flags — so pass the same
flags you generate with. The default is `asyncapi.yaml` in `dir`.

```bash
# fail CI when asyncapi.yaml is out of date (check verifies asyncapi.yaml only)
asyncgo check .
asyncgo check . --config ./ci/mappings.yaml
```

```console
$ asyncgo check .
/home/you/orders/asyncapi.yaml is up to date
```

| Exit code | Meaning |
| --- | --- |
| `0` | The artifact is up to date |
| `1` | The check could not run — a catalog build error, a missing artifact, an I/O failure |
| `2` | Usage error — an unknown flag, too many arguments (Cobra's own convention) |
| `3` | The artifact is out of date — regenerate it |

Splitting `1` from `3` is the point, so CI can tell "regenerate the committed
document" from "the build is broken":

```bash
asyncgo check . || case $? in
  3) echo "asyncapi.yaml is stale: run asyncgo generate ." ;;
  *) echo "the catalog does not build" ;;
esac
```

## Discovery and configuration

The generator discovers catalogs across the module's packages, then runs a small
harness to materialize them — it never executes your `main` package. Both commands
read schema mappings from `asyncgo.yaml` in the module root, or from the file
`--config` names; see [Custom types](custom-types.md#third-party-types-and-declared-mappings).

Scope is the module, not the import graph: a catalog is generated whether or not a
`main` package imports it (a `main` package imports no catalog at all ⇒ a warning,
not a missing catalog). Package `main` itself is excluded, because the harness
cannot import a program. A module with several catalogs merges them into one
document, and a key two catalogs both define is reported as a collision.

## Verbose logging

`-v` narrates what the command is doing to stderr; repeat it (`-vv`) for
per-package detail and a timing recap.

```console
$ asyncgo generate . -v
asyncgo: +0.000s config: no asyncgo.yaml found, using built-in mappings only
asyncgo: +0.000s load: loading packages in /home/you/orders...
asyncgo: +0.120s load: 2 packages, 2 imported by a main in 119.433ms
asyncgo: +0.120s catalogs: 1 catalog found in 15µs
asyncgo: +0.120s harness: materializing 1 catalog...
asyncgo: +0.370s harness: 1 catalog valid, 0 errors in 250.55ms
asyncgo: +0.371s merge: 1 channel, 1 server, 1 schema, 0 collisions in 41µs
asyncgo: +0.371s encode: yaml, 930 B in 768µs
asyncgo: +0.372s write: /home/you/orders/asyncapi.yaml in 211µs
wrote /home/you/orders/asyncapi.yaml (1 catalog(s))
```

`-vv` adds which catalog was found and where, the combinator reference count, the
generated harness path, and a per-stage recap:

```console
asyncgo: +0.081s catalogs: github.com/acme/orders.Catalog declared in catalog.go
asyncgo: +0.081s refs: 0 combinator refs from 0 packages, 0 alias edges
asyncgo: +0.081s harness: writing harness program to .../_asyncgo-harness-251665988/main.go
asyncgo: +0.206s descriptions: applied 1 field descriptions across 1 schemas
asyncgo: recap
  config        32µs
  load       80.907ms  2 packages, 2 imported by a main
  catalogs       5µs  1 catalog found
  harness   125.348ms  1 catalog valid, 0 errors
  merge         17µs  1 channel, 1 server, 1 schema, 0 collisions
  encode       520µs  yaml, 930 B
  write        146µs  /home/you/orders/asyncapi.yaml
  total     207.194ms
```

Logs go to stderr, so `asyncgo generate . -v 1>/dev/null` gives you just the
narrative and `2>/dev/null` just the result.
