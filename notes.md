## Foundations

**Basics**

- Handlers execute your application logic
- Servemux stores the mapping between URL routing patterns and the corresponding handlers
- Good practice for your module path to equal the location that the code can be downloaded from

Route pattern that ends with trailing slash known as _subtree path patern_. matched whenever start of request URL path matches subtree path. This explains why `"/"` is treated as subtree. Think of it like having a wildcard `"/path/**"`. Prevent subtree from acting like they have wildcard with `{$}`.

**Servemux Features**

Request URL paths are automatically sanitized. If request contains any `.` or `..` elements or repeated slashes, user is redirected to equivalent claen URL. `301 Permanent Redirect`.

A default servemux will be used if not explicitly created. Avoid this as it becomes a global variable at `http.DefaultServeMux` that can be modified anywhere in your code. A compromised third-party package could also register a malicious handler and expose it to the web from your application.

Each path segment can only contains one wildcard, and it must fill whole segment. `/{y}-{m}-{d}` and `/{slug}.html` are not valid, but `/{id}` is. Retrieve with `r.PathValue(value string)`

Go will automcatically send `405 Method Not Allowed` responses for unregistered routes.

**Precedence and conflicts**

It is possible that some patterns will conflict. The most specific route pattern wins in Go. For example, `foo/bar` will beat `/foo/{id}`. Equally specific patterns such as `/post/new/{id}` and `/post/{author}/latest` will cause a runtime panic when initializing the routes.

**Remainder Wildcards**

A wildcard identifier ending with `...` will match any and all remaining segments of a request path. This is similar to how a pattern ending in a `/` will match all subtrees, except that `/foo/{path...}` notation allows you to access the remaining segments with `r.PathValue("path")`.

**More**

`w.WriteHeader()` can only be called once per response. If not called, the first call to `w.Write()` will automatically send a `200` status code.

Customize headers with `w.Header().Add(key, value string)`. Make sure it is called before `w.WriteHeader()` or `w.Write()`, will have no effect otherwise. Also see `Set()`, `Del()`, `Get()` and `Values()`.

Go does content sniffing with the `http.DetectContentType()` function. Will fallback on `Content-Type: application/octet-stream` if it cannot guess. One problem is that it can't distingush between JSON and plain text. So, by default, JSON responses will be sent with a `Content-Type: text/plain; charset=utf-8`. Do the following to fix that.

```go
w.Header().Set("Content-Type", "application/json")
w.Write([]byte(`{"name":"Alex"}`))
```

**Structure**

- `cmd` will contain application-specific for the executable applications in the project.
- `internal` will contain ancillary non-application-specific code used in project. This directory names carries a special meaning and behaviour in Go. Any packages inside the `internal` can only imported by the parent of `internal` - our entire project in this case. This prevents other codebases from importing and relying onpackages from our `internal` directory
- `ui` will contain UI assets.

**HTML Templating**

```
{{define "base"}}
{{template "nav" .}}
{{define "title"}}Home{{end}}
```

```go
template.ParseFiles(filename ...string)
ts.ExecuteTemplate(wr io.Writer, name string, data any)
```

**Serving Static Files**

```go
fileServer := http.fileServer(http.Dir("ui/static"))
mux.Handle("GET /static/", http.StripPrefix("/static", fileServer))
```

`http.FileServer`

- Sanitizes all request paths by running them through `path.Clean()`
- Range requests are fully supported
- `Content-Type` is automatically set from file extension

Important to note that `http.FileServer` probably won’t be reading these files from disk once the application is up-and-running. Both Windows and Unix-based operating systems cache recently-used files in RAM, so (for frequently-served files at least) it’s likely that http.FileServer will be serving them from RAM rather than making the relatively slow round-trip to your hard disk.

- Could serve a single file with `http.ServeFile(w, r, "ui/file.zip")`. Does not sanitize file path if constructing file path from unstrusted user input.

https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings

**Handler Interface**

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

Requests are handled concurrently. All incoming HTTP requests are served in their own goroutine. The downside is that you need to be aware of (and protect against) race conditions when accessing shared resources from your handlers.

## Configuration and error handling

> Command-line flags, loggers, dependencies, centralize err handling

## CL Flags

```go
// go run ./cmd/web -addr=":4000"

addr := flag.String("addr", ":4000", "HTTP network address")
flag.Parse()
```

There are also flgs for ints, bools, duration, etc. If you want tp read the falg into the memory address of an existing variable, use the `StringVar()` variation.

```go
type config struct {
    addr      string
}

...

var cfg config

flag.StringVar(&cfg.addr, "static-dir", "./ui/static", "Path to static assets")
flag.Parse()
```

**Logging**

- Concurrency-safe

```go
// Different ways to log kv pairs
logger.Info("starting server", "addr", *addr)
logger.Info("starting server", slog.Any("addr", *addr))
logger.Info("starting server", slog.String("addr", *addr))

```

## Database-driven Responses

```bash
brew install mysql

brew services start mysql
mysql -u root -p

# login as new user
mysql -D snippety -u web -p
```

```sql
CREATE DATABASE snippety CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE snippety;

CREATE USER 'web'@'localhost';
GRANT SELECT, INSERT, UPDATE, DELETE ON snippety.* TO 'web'@'localhost';
ALTER USER 'web'@'localhost' IDENTIFIED BY 'math';

exit
```

- `go.sum` contains all cryptographic checksums representing the content of require packages
- `go mod verify` will verify that the checksums of the downloaded packages on your machine match the entries in go.sum
- `go mod download` will download all dependencies for project

```go
db, err := sql.Open("mysql", "web:pass@/snippetbox?parseTime=true")
```

- `mysql`: driver name
- `web:pass@/snippetbox?parseTime=true`: data source name (DSN)
- `parseTime=true`: driver-specific param which instruct driver to convert SQL time and data fields to Go's time.Time objects
- `sql.Open()` opens an `sql.DB` object - a pool of many connections, actual connections are established lazily. Safe for concurrent access. Intended to be long lived. Should not be called in short-lived HTTP handlers - waste of memory and network resources.

**nitty**

- `DB.Query()` is used for SELECT queries which return multiple rows.
- `DB.QueryRow()` is used for SELECT queries which return a single row.
- `DB.Exec()` is used for statements which don’t return rows (like INSERT and DELETE).

`sql.Result` type returned by `DB.Exec()` provides two methods:

1. `LastInsertId()` — which returns the integer (an int64) generated by the database in response to a command.
2. `RowsAffected()` — which returns the number of rows (as an int64) affected by the statement.

> Not all drivers supports these two methods

**Placeholder Parameters**

- Using `?` helps to construct a quiery helps avoid injection attacks from any untrusted user-provided input.
- Behind the scenes, `DB.Exec()` works in three steps.
  1. Creates new prepared statement. Db parses, compiles, and stores it ready for execution.
  2. Passes param values to database. Execute statement using params.
  3. Closes/deallocates prepared statement on db.
- Postgres uses `$N` notation

**More**

- Transactions
- Prepared statements
-

## HTML

html/template package automatically escapes any data that is yielded between `{{ }}` tags. This behavior is hugely helpful in avoiding cross-site scripting (XSS) attacks, and is the reason that you should use the html/template package instead of the more generic text/template package that Go also provides.

If the type that you’re yielding between {{ }} tags has methods defined against it, you can call these methods:

```html
<!-- Calls AddDate on a Go time.Time type with params 0 6 0 -->
<span>{{.Snippet.Created.AddDate 0 6 0}}</span>
```

## Template Actions

```html
{{ for, if, eq, and, len, etc... }}
```
