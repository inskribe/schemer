# Schemer

**Schemer** is a lightweight, delta-based database migration tool for PostgreSQL.  

---

## 🚀 Features

- Versioned `.sql` delta files (`up`, `down`, and optional `post`)
- Apply deltas in order or cherry-pick specific versions
- Post-delta support for deferred cleanup
- Schema state tracked via a dedicated `schemer` table
- Simple `.env`-based configuration `optional`
- Built with Go — no runtime dependencies

---

## 📦 Installation

```sh
go install github.com/yourusername/schemer@latest
```

Ensure `$GOPATH/bin` is in your system `PATH`.

---

## 📁 Delta Format

Deltas live in the `deltas/` directory and follow this naming convention:

```
<version>_<name>.<type>.sql
```

- `<version>` — zero-padded integer (e.g. `001`)
- `<name>` — descriptive name (e.g. `add_users`)
- `<type>` — one of `up`, `down`, or `post`

### Example

```
deltas/
├── 001_add_users.up.sql
├── 001_add_users.down.sql
├── 001_add_users.post.sql  # optional
```

---

## 🛠 Commands

### `schemer init`

Initializes a new project in the current directory:

- Creates a `deltas/` directory
- Generates a `.env` file with `DATABASE_URL`
- Writes a `schemer.sql` template

``` sh
Should only be used on new projects and never in a production
environment.
```

---

### `schemer create <name> [--post]`

Creates a versioned delta group:

```
schemer create add_users
schemer create cleanup_sessions --post
```

This creates:

```
deltas/002_add_users.up.sql
deltas/002_add_users.down.sql
deltas/002_add_users.post.sql  # only if --post is used
```

---

### `schemer up [options]`

Default behaviour: Applies all unapplied `up` deltas.

**Options:**

- `--from <tag>` — start from a specific tag
- `--to <tag>` — apply up to a tag
- `--cherry-pick <tag> <tag>` — apply specific tags
- `--prune` — skip no-op deltas

---

### `schemer down [options]`

Default behaviour: Rolls back last applied delta.

**Options:**

- `--from <tag>` — rollback from this tag
- `--to <tag>` — rollback down to this tag
- `--cherry-pick <tag> <tag>` — rollback specific tags
- `--prune` — skip no-op deltas

---

### `schemer post [options]`

Default behaviour: Applies all `post` deltas for all recorded `up` deltas.

**Options:**

- `--from <tag>` / `--to <tag>` — limit range
- `--cherry-pick <tag> <tag>` — apply specific posts
- `--force` — apply untracked post deltas

---

## 🧪 Examples

```sh
schemer init
schemer create create_users --post
schemer up
schemer down --from 003
schemer post --cherry-pick 003 --force
```

---

## 📄 .env File

In a dev environment Schemer reads the connection string from `.env`:

```env
DATABASE_URL=postgres://user:pass@localhost:5432/mydb?sslmode=disable
```

---

## 📊 Schemer Table

A `schemer` table tracks:

- Applied delta tags
- Post delta status

It's created during `init` or the first migration.

