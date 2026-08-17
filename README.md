# quaere

A terminal SQL client with a [Beekeeper Studio](https://www.beekeeperstudio.io/)-style
layout, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Saved
connections and their tables live in a left sidebar; the main area holds a query
editor on top and a scrollable results grid below.

```
╭──────────────╮╭────────────────────────────────────────╮
│ Database     ││ Query                                  │
│ CONNECTIONS  ││   1 SELECT * FROM "public"."users" …   │
│ ● local-app  │╰────────────────────────────────────────╯
│   prod       │╭────────────────────────────────────────╮
│ TABLES       ││ Results   SELECT 3 · 3 rows · 3ms      │
│   users      ││  id  name    email                     │
│   orders     ││  1   Ada     ada@x.com                 │
│              ││  2   Alan    alan@x.com                │
╰──────────────╯╰────────────────────────────────────────╯
```

quaere speaks **PostgreSQL**, via either method:

- **Connection string** — a standard `postgres://user:pass@host:port/dbname` URL.
- **Cloud SQL (IAM)** — a Google Cloud SQL for PostgreSQL instance authenticated
  with IAM database auth over Application Default Credentials. No password is
  stored: your IAM identity is the credential. If no credentials are present,
  quaere runs `gcloud auth application-default login` (opening a browser) for you.

## Install

```sh
brew install aurelio-labs/quaere/quaere
```

Or build from source (requires Go 1.25+):

```sh
git clone https://github.com/aurelio-labs/quaere
cd quaere
go build -o quaere .
```

For the Cloud SQL IAM method you also need the [gcloud CLI](https://cloud.google.com/sdk/docs/install)
installed, and the target instance must have [IAM database authentication](https://cloud.google.com/sql/docs/postgres/iam-authentication)
enabled with your principal added as a database user.

## Usage

```sh
quaere
```

On first run the sidebar is empty — press `n` to add a connection.

| Key            | Action                                             |
| -------------- | -------------------------------------------------- |
| `tab`          | switch pane (sidebar → query → results)            |
| `↑` / `↓`      | move within the focused pane                        |
| `enter`        | open a connection / run a highlighted table         |
| `n`            | new connection                                      |
| `e`            | edit the selected connection                        |
| `d`            | delete the selected connection                      |
| `ctrl+r`       | run the query in the editor                          |
| `q`            | quit (except while typing in the editor)             |
| `ctrl+c`       | quit from anywhere                                   |

In the connection form: `tab`/`↑`/`↓` move between fields, `←`/`→` toggle the
method, `enter` advances (and saves on the last field), `ctrl+s` saves, `esc`
cancels.

Selecting a table generates `SELECT * FROM <table> LIMIT 100;` in the editor and
runs it, so you can inspect any table with a single keystroke.

## Configuration

Saved connections are stored as JSON at:

```
$XDG_CONFIG_HOME/quaere/connections.json   # defaults to ~/.config/quaere/
```

Connection strings may embed a password, so the file is written owner-only
(`0600`). The Cloud SQL IAM method stores no secrets — only the instance name,
IAM user, and database.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

An end-to-end test against a real PostgreSQL is included but skipped unless a DSN
is provided:

```sh
docker run -d --name pg -e POSTGRES_PASSWORD=secret -e POSTGRES_DB=app -p 55432:5432 postgres:16
QUAERE_PG_DSN='postgres://postgres:secret@localhost:55432/app?sslmode=disable' \
  go test ./internal/tui -run Live -v
```

## License

MIT
