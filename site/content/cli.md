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
| `-o`, `--output` | Output file, **or a directory** (trailing separator) to write one document per catalog into. Defaults to `<dir>/asyncapi.<format>` |
| `--config` | Schema-mapping file. Defaults to `<dir>/asyncgo.yaml` |

```bash
# write asyncapi.yaml (committed artifact)
asyncgo generate .

# write asyncapi.json instead (2-space indented, newline-terminated)
asyncgo generate . --format json

# write to a custom file
asyncgo generate . -o ./docs/asyncapi.yaml

# one document per catalog, into a directory
asyncgo generate . -o ./docs/

# an explicit -o file wins whatever its extension: --format picks the encoding
asyncgo generate . -o ./docs/spec.yaml --format json

# read schema mappings from a file other than <dir>/asyncgo.yaml
asyncgo generate . --config ./ci/mappings.yaml
```

`-o` names the destination, `--format` picks the encoding: `-o spec.yaml --format json`
writes JSON to `spec.yaml`. A directory passed to `-o` must already exist.

### One document per catalog

A **file** destination writes one merged document for the whole module — every
catalog in it, with the first definition winning where two catalogs define the
same key. That is the default and the right shape when the module describes one
topology.

A **directory** destination writes one document *per catalog*, inside it:

```console
$ asyncgo generate . -o docs/
wrote /home/you/orders/docs/orders.asyncapi.yaml
wrote /home/you/orders/docs/payments.asyncapi.yaml
```

The filename carries the catalog's document name:

| Name | Filename |
| --- | --- |
| `<package>.<variable>` (derived default) | `orders.Catalog.asyncapi.yaml` |
| `.Named("orders")` | `orders.asyncapi.yaml` |

The `asyncapi` infix is what lets `check` tell its own artifacts from any other
YAML in the directory. Each document is built from its own catalog alone, so it
keeps its own `info` — the merge, by contrast, takes `info` from the first
catalog that defines a title and drops the rest.

Give a catalog an explicit name so the filename does not depend on where the
variable lives:

```go
var Orders = asyncgo.Spec(
	asyncgo.Info("Orders Service", "1.0.0"),
	// ...
).Named("orders")
```

The name must match `^[\w\d.\-_]+$`, which is also what makes it safe as a
filename. Two catalogs resolving to the same name are an error, and nothing is
written: the whole set is validated and encoded before the first file.

!!! warning "A trailing separator selects the mode"

    `-o docs` and `-o docs/` are different: the first names a file, the second a
    directory. Directory mode prints one `wrote` line per file, so the mode a run
    actually took is visible at default verbosity.

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

With a directory destination it checks the whole artifact set — the same
one-document-per-catalog files `generate . -o docs/` would write:

```console
$ asyncgo check . -o docs/
/home/you/orders/docs: 2 document(s) up to date
```

One verdict covers the set, so CI keeps branching on a single code. Three failure
classes report as exit 3, and `-v` names each one:

| Class | What it means |
| --- | --- |
| **stale** | The file differs from what the catalog generates now |
| **missing** | A catalog has no committed document yet — it was added since the last `generate` |
| **orphan** | A `*.asyncapi.<format>` file no catalog claims — its catalog was deleted or renamed |

An orphan is why the filenames carry the `asyncapi` infix: only files matching
`*.asyncapi.<format>` are adopted, so an unrelated `docs/architecture.yaml` is
never mistaken for a stale document, and an artifact whose catalog is gone is
never silently unverified.

| Exit code | Meaning |
| --- | --- |
| `0` | The artifact(s) are up to date |
| `1` | The check could not run — a catalog build error, an I/O failure, or (file mode only) a missing artifact |
| `2` | Usage error — an unknown flag, too many arguments (Cobra's own convention) |
| `3` | The artifact set is out of date — regenerate it |

A missing artifact is exit 1 only in **file** mode, where you named the one path
to compare and its absence is a setup mistake. In directory mode the file set is
derived by `asyncgo`, so an absent member is a statement about the committed set
and reports as 3.

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
cannot import a program.

A module with several catalogs is written as one merged document by default, and
a key two catalogs both define is reported as a collision. Pass a directory to
`-o` to get one document per catalog instead — see
[one document per catalog](#one-document-per-catalog).

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
