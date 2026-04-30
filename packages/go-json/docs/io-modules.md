# I/O Modules

I/O modules give go-json programs side-effect capabilities — HTTP requests, file
system access, database queries, shell commands, and more. They are **opt-in at
two levels** so that untrusted programs run in a pure sandbox by default.

## Opt-in Model

### 1. Program must import the module

```json
{
  "import": {
    "http": "io:http",
    "fs": "io:fs"
  }
}
```

### 2. Runtime must enable the module

```go
rt := gojson.NewRuntime(gojson.WithIO(goio.HTTP(), goio.FS()))
```

The two levels are enforced independently:

| Imported? | Enabled? | Result |
|-----------|----------|--------|
| Yes | Yes | Works |
| Yes | No | **Compile error** — module not available |
| No | Yes | **Symbol not found** — program never references it |
| No | No | Nothing happens (default) |

This means a host can safely enable modules without worrying about programs that
don't use them, and a program that declares its dependencies will fail fast if
the host hasn't opted in.

### Three Ways to Call I/O Functions

I/O module functions can be called three ways. Choose based on whether your arguments are literal data or computed expressions:

```json
// call + args — literal values, no escaping needed
{"let": "resp", "call": "http.get", "args": ["https://api.example.com/users"]}
{"call": "fs.write", "args": ["./log.txt", "Hello, World!"]}

// call + with (array) — expression args, variables evaluated
{"let": "resp", "call": "http.get", "with": ["url"]}
{"call": "fs.write", "with": ["'./log.txt'", "content"]}

// expr — inline expression, good for one-liners
{"let": "resp", "expr": "http.get('https://api.example.com/users')"}
{"let": "_", "expr": "fs.write('./log.txt', content)"}
```

**Fire-and-forget** (no return value needed) — use `call` directly, no throwaway variable:

```json
{"call": "fs.write", "args": ["./output.txt", "done"]}
{"call": "redis.set", "args": ["key", "value"]}
```

See [Language Reference — Three Ways to Call Functions](language-reference.md#three-ways-to-call-functions) for the full explanation.

---

## HTTP Module (`io:http`)

**Import:** `"http": "io:http"`
**Enable:** `goio.HTTP()`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `get` | `http.get(url, headers?, timeout?, auth?)` | GET request |
| `post` | `http.post(url, body?, headers?, timeout?, auth?)` | POST request |
| `put` | `http.put(url, body?, headers?, timeout?, auth?)` | PUT request |
| `patch` | `http.patch(url, body?, headers?, timeout?, auth?)` | PATCH request |
| `delete` | `http.delete(url, headers?, timeout?, auth?)` | DELETE request |

### Examples

**Simple GET** — three ways:

```json
// args — URL is literal
{"let": "resp", "call": "http.get", "args": ["https://api.example.com/users"]}

// with — URL from variable
{"let": "resp", "call": "http.get", "with": ["apiUrl"]}

// expr — inline
{"let": "resp", "expr": "http.get('https://api.example.com/users')"}
```

**POST with body:**

```json
// args — body is literal JSON object
{"let": "resp", "call": "http.post", "args": [
  "https://api.example.com/users",
  {"name": "Alice", "age": 30}
]}

// with — body from variable
{"let": "resp", "call": "http.post", "with": ["apiUrl", "userData"]}

// expr — inline
{"let": "resp", "expr": "http.post('https://api.example.com/users', {'name': 'Alice', 'age': 30})"}
```

**GET with custom headers and timeout:**

```json
// args — all literal
{"let": "resp", "call": "http.get", "args": [
  "https://api.example.com/data",
  {"X-Custom": "value"},
  5000
]}

// expr — inline
{"let": "resp", "expr": "http.get('https://api.example.com/data', {'X-Custom': 'value'}, 5000)"}
```

**Authenticated request (Bearer):**

```json
// args — token is literal
{"let": "resp", "call": "http.get", "args": [
  "https://api.example.com/me",
  {},
  null,
  {"type": "bearer", "token": "eyJhbG..."}
]}

// with — token from variable
{"let": "resp", "call": "http.get", "with": [
  "'https://api.example.com/me'", "{}", "nil",
  "{'type': 'bearer', 'token': authToken}"
]}
```

**Authenticated request (Basic):**

```json
{"let": "resp", "call": "http.post", "args": [
  "https://api.example.com/login",
  {},
  {},
  null,
  {"type": "basic", "username": "admin", "password": "s3cret"}
]}
```

### Response Shape

Every HTTP function returns an object with this structure:

```json
{
  "status": 200,
  "body": { "id": 1, "name": "Alice" },
  "headers": {
    "content-type": "application/json",
    "x-request-id": "abc-123"
  }
}
```

- **`status`** — HTTP status code (integer).
- **`body`** — Parsed JSON if the response `Content-Type` is JSON. Otherwise returned as a raw string.
- **`headers`** — Response headers as a flat object (keys lowercased).

### Behavior Notes

- Redirects are followed automatically, up to **10 hops**.
- Response body is truncated at **MaxResponseSize** (default 10 MB).
- Non-JSON response bodies are returned as a string.
- The cloud metadata endpoint `169.254.169.254` is blocked by default (see [Security Configuration](#security-configuration)).

### Auth Parameter

| Type | Shape |
|------|-------|
| Bearer | `{"type": "bearer", "token": "..."}` |
| Basic | `{"type": "basic", "username": "...", "password": "..."}` |

---

## FS Module (`io:fs`)

**Import:** `"fs": "io:fs"`
**Enable:** `goio.FS()`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `read` | `fs.read(path, encoding?)` | Read file contents |
| `write` | `fs.write(path, content)` | Create/overwrite file |
| `append` | `fs.append(path, content)` | Append to file (creates if missing) |
| `exists` | `fs.exists(path)` | Check if path exists → `bool` |
| `list` | `fs.list(path, detail?)` | List directory contents |
| `mkdir` | `fs.mkdir(path)` | Create directory (recursive) |
| `remove` | `fs.remove(path)` | Delete file or directory |
| `stat` | `fs.stat(path)` | File/directory metadata |
| `copy` | `fs.copy(source, destination)` | Copy file |
| `move` | `fs.move(source, destination)` | Move/rename file |
| `glob` | `fs.glob(pattern)` | Pattern-match filenames |

### Examples

**Read a file:**

```json
// args — path is literal
{"let": "content", "call": "fs.read", "args": ["./config.json"]}

// with — path from variable
{"let": "content", "call": "fs.read", "with": ["filePath"]}

// expr — inline
{"let": "content", "expr": "fs.read('./config.json')"}
```

**Read binary as base64:**

```json
{"let": "imageData", "call": "fs.read", "args": ["./logo.png", "base64"]}
{"let": "imageData", "expr": "fs.read('./logo.png', 'base64')"}
```

**Write a file:**

```json
// args — content is literal (no escaping needed for special chars)
{"call": "fs.write", "args": ["./output.txt", "Hello, World!"]}

// with — content from variable
{"call": "fs.write", "with": ["'./output.txt'", "result"]}

// expr
{"let": "_", "expr": "fs.write('./output.txt', result)"}
```

**Append to a log:**

```json
{"call": "fs.append", "with": ["'./app.log'", "logLine + '\\n'"]}
{"call": "fs.append", "args": ["./app.log", "a static log line\n"]}
```

**Check existence and read conditionally:**

```json
[
  { "let": "hasConfig", "expr": "fs.exists('./config.json')" },
  {
    "if": "hasConfig",
    "then": { "let": "config", "expr": "fs.read('./config.json')" },
    "else": { "let": "config", "expr": "'{}'" }
  }
]
```

**List directory (simple):**

```json
{
  "let": "files",
  "expr": "fs.list('./data')"
}
```

Returns: `["file1.json", "file2.json", "subdir"]`

**List directory (detailed):**

```json
{
  "let": "files",
  "expr": "fs.list('./data', true)"
}
```

Returns:

```json
[
  { "name": "file1.json", "size": 1024, "modified": "2024-01-15T10:30:00Z" },
  { "name": "subdir", "size": 0, "modified": "2024-01-14T08:00:00Z" }
]
```

**File metadata:**

```json
{
  "let": "info",
  "expr": "fs.stat('./data/report.csv')"
}
```

Returns: `{"name": "report.csv", "size": 4096, "modified": "2024-01-15T10:30:00Z", "is_dir": false}`

**Glob pattern matching:**

```json
{
  "let": "jsonFiles",
  "expr": "fs.glob('./data/*.json')"
}
```

### Security

- All paths are validated against the `SecurityConfig` (allowed/blocked path lists).
- Symlinks are resolved to their real target, then re-checked against the path rules.
- Path traversal (`../../`) is resolved to an absolute path and checked — you cannot escape the allowed directories.
- On Windows, path matching is case-insensitive.

---

## SQL Module (`io:sql`)

**Import:** `"sql": "io:sql"`
**Enable:** `goio.SQL()`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `query` | `sql.query(sql, params?)` | SELECT — returns rows |
| `execute` | `sql.execute(sql, params?)` | INSERT/UPDATE/DELETE — returns affected count |
| `begin` | `sql.begin()` | Start a transaction |
| `commit` | `sql.commit()` | Commit the current transaction |
| `rollback` | `sql.rollback()` | Roll back the current transaction |

### Examples

**Query with positional parameters:**

```json
{
  "let": "users",
  "expr": "sql.query('SELECT * FROM users WHERE age > ?', [18])"
}
```

Returns:

```json
{
  "rows": [
    { "id": 1, "name": "Alice", "age": 30 },
    { "id": 2, "name": "Bob", "age": 25 }
  ],
  "columns": ["id", "name", "age"],
  "count": 2
}
```

**Query with named parameters:**

```json
{
  "let": "users",
  "expr": "sql.query('SELECT * FROM users WHERE name = :name AND age > :minAge', {'name': 'Alice', 'minAge': 18})"
}
```

**Execute (INSERT):**

```json
{
  "let": "result",
  "expr": "sql.execute('INSERT INTO users (name, age) VALUES (?, ?)', ['Alice', 30])"
}
```

Returns: `{"rows_affected": 1, "last_insert_id": 42}`

**Transaction:**

```json
[
  {"let": "_", "expr": "sql.begin()"},
  {
    "let": "r1",
    "expr": "sql.execute('UPDATE accounts SET balance = balance - ? WHERE id = ?', [100, 1])"
  },
  {
    "let": "r2",
    "expr": "sql.execute('UPDATE accounts SET balance = balance + ? WHERE id = ?', [100, 2])"
  },
  {
    "if": "r1.rows_affected == 1 && r2.rows_affected == 1",
    "then": [{"let": "_", "expr": "sql.commit()"}],
    "else": [{"let": "_", "expr": "sql.rollback()"}]
  }
]
```

### Supported Drivers

| Driver | Auto-detected from DSN |
|--------|------------------------|
| SQLite | File path or `:memory:` |
| PostgreSQL | `postgres://` or `postgresql://` |
| MySQL | `mysql://` or `user:pass@tcp(host)/db` |
| SQL Server | `sqlserver://` or `mssql://` |
| Oracle | `oracle://` |

### Unified Parameter Syntax

Write `?` (positional) or `:name` (named) in your queries. The engine
auto-translates to the driver's native syntax:

| Your query | PostgreSQL | SQL Server | MySQL | SQLite |
|------------|-----------|------------|-------|--------|
| `?` | `$1`, `$2` | `@p1`, `@p2` | `?` | `?` |
| `:name` | `$1` (ordered) | `@name` | `?` (ordered) | `:name` |

### Connection Modes

- **Standalone mode:** The `dsn` parameter is required in the module configuration. The engine manages its own connection pool per DSN.
- **Hosted mode:** The host application provides the database connection. No DSN needed in the program.

### Behavior Notes

- Connection pooling is managed per-DSN.
- Transactions support savepoints for nested transaction semantics.
- DDL statements are blocked by default (configurable via `BlockedKeywords`).
- Maximum query length is enforced by the security config.

---

## Exec Module (`io:exec`)

**Import:** `"exec": "io:exec"`
**Enable:** `goio.Exec()`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `run` | `exec.run(command, args?, env?, timeout?)` | Execute a system command |

### Examples

**Simple command:**

```json
{
  "let": "result",
  "expr": "exec.run('ls', ['-la', '/tmp'])"
}
```

Returns:

```json
{
  "exit_code": 0,
  "stdout": "total 48\ndrwxrwxrwt 12 root root ...",
  "stderr": ""
}
```

**Command with custom environment:**

```json
{
  "let": "result",
  "expr": "exec.run('node', ['script.js'], {'NODE_ENV': 'production', 'PORT': '3000'})"
}
```

**Command with timeout (milliseconds):**

```json
{
  "let": "result",
  "expr": "exec.run('ping', ['-c', '3', 'example.com'], null, 5000)"
}
```

**Checking exit code:**

```json
[
  { "let": "result", "expr": "exec.run('grep', ['-r', 'TODO', './src'])" },
  {
    "if": "result.exit_code == 0",
    "then": { "return": "result.stdout" },
    "else": { "return": "'No TODOs found'" }
  }
]
```

### Security

**Command whitelist:** Only commands listed in `AllowedCommands` can be executed. If the whitelist is empty, no commands are allowed.

**Permanently denied commands** (blocked regardless of whitelist):

`rm`, `rmdir`, `del`, `format`, `shutdown`, `reboot`, `halt`, `poweroff`, `dd`, `mkfs`, `fdisk`

**No shell expansion:** Arguments are passed as an array directly to the process. There is no shell involved — no pipes (`|`), no redirects (`>`), no globbing (`*`), no command chaining (`&&`, `;`).

**Environment variables:**
- If the `env` parameter is provided, **only** those variables are available to the child process.
- If `env` is omitted, the host's environment is inherited **minus** engine secrets (`JWT_SECRET`, `DB_PASSWORD`, `ENCRYPTION_KEY`, `SMTP_PASSWORD`, `STORAGE_S3_SECRET_KEY`, `STORAGE_S3_ACCESS_KEY`).

**Output:** Truncated at `MaxOutputSize` (default 1 MB).

**Exit codes:** A non-zero exit code is **not** treated as an error. It is returned in `exit_code` for the program to handle.

---

## MongoDB Module (`io:mongo`)

> **Status: Functional** — Uses `MongoDriver` interface with built-in `InMemoryMongoDriver` for development/testing. For production, inject a real driver via `WithMongoDriver(driver)` (e.g., wrapping `go.mongodb.org/mongo-driver/v2`).

**Import:** `"mongo": "io:mongo"`
**Enable:** `goio.Mongo()`

### Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `find` | `mongo.find(collection, filter?, options?)` | Find documents |
| `findOne` | `mongo.findOne(collection, filter?)` | Find single document |
| `insert` | `mongo.insert(collection, document)` | Insert one document |
| `insertMany` | `mongo.insertMany(collection, documents)` | Insert multiple documents |
| `update` | `mongo.update(collection, filter, update)` | Update matching documents |
| `delete` | `mongo.delete(collection, filter)` | Delete matching documents |
| `count` | `mongo.count(collection, filter?)` | Count matching documents |
| `aggregate` | `mongo.aggregate(collection, pipeline)` | Aggregation pipeline |

### Examples

**Find with filter and options:**

```json
{
  "let": "users",
  "expr": "mongo.find('users', {'active': true}, {'limit': 10, 'sort': {'created': -1}})"
}
```

**Find one:**

```json
{
  "let": "user",
  "expr": "mongo.findOne('users', {'_id': userId})"
}
```

**Insert:**

```json
{
  "let": "result",
  "expr": "mongo.insert('users', {'name': 'Alice', 'age': 30, 'active': true})"
}
```

**Insert many:**

```json
{
  "let": "result",
  "expr": "mongo.insertMany('logs', [{'event': 'login', 'ts': now()}, {'event': 'pageview', 'ts': now()}])"
}
```

**Update:**

```json
{
  "let": "result",
  "expr": "mongo.update('users', {'_id': userId}, {'$set': {'active': false}})"
}
```

**Aggregation pipeline:**

```json
{
  "let": "stats",
  "expr": "mongo.aggregate('orders', [{'$group': {'_id': '$status', 'total': {'$sum': '$amount'}}}])"
}
```

### Security

- **NoSQL injection protection:** `$where` and `$function` operators are blocked.
- **Security config:** `AllowedDatabases`, `MaxDocumentSize`, `MaxResults`.

---

## Redis Module (`io:redis`)

> **Status: Functional** — Uses `RedisDriver` interface with built-in `InMemoryRedisDriver` for development/testing. For production, inject a real driver via `WithRedisDriver(driver)` (e.g., wrapping `github.com/redis/go-redis/v9`).

**Import:** `"redis": "io:redis"`
**Enable:** `goio.Redis()`

### Functions

#### Strings

| Function | Signature | Description |
|----------|-----------|-------------|
| `get` | `redis.get(key)` | Get value by key |
| `set` | `redis.set(key, value, ttl?)` | Set value with optional TTL (seconds) |
| `del` | `redis.del(key)` | Delete a key |
| `exists` | `redis.exists(key)` | Check if key exists → `bool` |
| `expire` | `redis.expire(key, seconds)` | Set TTL on existing key |
| `ttl` | `redis.ttl(key)` | Get remaining TTL (seconds) |
| `incr` | `redis.incr(key)` | Increment integer value |
| `decr` | `redis.decr(key)` | Decrement integer value |

#### Hashes

| Function | Signature | Description |
|----------|-----------|-------------|
| `hget` | `redis.hget(key, field)` | Get hash field |
| `hset` | `redis.hset(key, field, value)` | Set hash field |
| `hgetall` | `redis.hgetall(key)` | Get all hash fields → object |

#### Lists

| Function | Signature | Description |
|----------|-----------|-------------|
| `lpush` | `redis.lpush(key, value)` | Push to head of list |
| `rpush` | `redis.rpush(key, value)` | Push to tail of list |
| `lrange` | `redis.lrange(key, start, stop)` | Get range from list |

#### Sets

| Function | Signature | Description |
|----------|-----------|-------------|
| `sadd` | `redis.sadd(key, member)` | Add member to set |
| `smembers` | `redis.smembers(key)` | Get all set members |

#### Pub/Sub

| Function | Signature | Description |
|----------|-----------|-------------|
| `publish` | `redis.publish(channel, message)` | Publish message to channel |

### Examples

**Get/Set with TTL:**

```json
[
  {
    "let": "_", "expr": "redis.set('user:123', userData, 3600)"
  },
  {
    "let": "cached",
    "expr": "redis.get('user:123')"
  }
]
```

**Counter:**

```json
[
  { "let": "count", "expr": "redis.incr('page:views:home')" },
  { "let": "_", "expr": "redis.expire('page:views:home', 86400)" }
]
```

**Hash (user profile):**

```json
[
  { "let": "_", "expr": "redis.hset('user:123', 'name', 'Alice')" },
  { "let": "_", "expr": "redis.hset('user:123', 'email', 'alice@example.com')" },
  { "let": "profile", "expr": "redis.hgetall('user:123')" }
]
```

**List (job queue):**

```json
[
  { "let": "_", "expr": "redis.rpush('jobs:pending', jobData)" },
  { "let": "pending", "expr": "redis.lrange('jobs:pending', 0, -1)" }
]
```

### Behavior Notes

- Non-string values are **automatically JSON serialized** on write and **deserialized** on read.
- `KeyPrefix` can be configured for tenant isolation (e.g., `"tenant1:"` → all keys prefixed automatically).

### Security

**Blocked commands** (always denied regardless of configuration):

`FLUSHALL`, `FLUSHDB`, `CONFIG`, `DEBUG`, `SHUTDOWN`, `SLAVEOF`, `REPLICAOF`

---

## Security Configuration

All I/O modules share a unified security configuration that the host provides at
runtime. Hardcoded deny lists are **always enforced** and cannot be overridden.

### SecurityConfig Struct

```go
type SecurityConfig struct {
    EnabledModules  []string

    HTTP   HTTPSecurityConfig
    FS     FSSecurityConfig
    SQL    SQLSecurityConfig
    Exec   ExecSecurityConfig
    Mongo  MongoSecurityConfig
    Redis  RedisSecurityConfig
}
```

### Per-Module Security

#### HTTP (`HTTPSecurityConfig`)

| Field | Type | Description |
|-------|------|-------------|
| `AllowedHosts` | `[]string` | Whitelist of allowed hostnames. Empty = all non-blocked allowed. Explicitly allowed hosts override BlockedHosts. |
| `BlockedHosts` | `[]string` | Blacklist (skipped for explicitly allowed hosts). |
| `MaxResponseSize` | `int64` | Max response body size in bytes. Default: 10 MB. |
| `Timeout` | `time.Duration` | Request timeout. |

**Hardcoded:** `169.254.169.254` (cloud metadata endpoint) is always blocked.

#### FS (`FSSecurityConfig`)

| Field | Type | Description |
|-------|------|-------------|
| `AllowedPaths` | `[]string` | Directories the program can access. |
| `BlockedPaths` | `[]string` | Directories always denied (takes precedence). |
| `MaxFileSize` | `int64` | Max file size for read/write operations. |
| `AllowWrite` | `bool` | Whether write/append/remove/move operations are permitted. |

**Hardcoded:** Path traversal (`../../`) is resolved to absolute paths and re-checked. Symlinks are resolved to their real target and re-validated. Windows paths use case-insensitive matching.

#### SQL (`SQLSecurityConfig`)

| Field | Type | Description |
|-------|------|-------------|
| `AllowedDrivers` | `[]string` | Which database drivers are permitted. |
| `MaxQueryTime` | `time.Duration` | Maximum query execution time. |
| `MaxRows` | `int` | Maximum rows returned per query. |
| `BlockedKeywords` | `[]string` | SQL keywords to block (e.g., `DROP`, `ALTER`, `TRUNCATE`). |

#### Exec (`ExecSecurityConfig`)

| Field | Type | Description |
|-------|------|-------------|
| `AllowedCommands` | `[]string` | Whitelist of executable commands. Empty = none allowed. |
| `MaxTimeout` | `time.Duration` | Maximum execution time per command. |
| `MaxOutputSize` | `int64` | Max stdout/stderr size. Default: 1 MB. |

**Hardcoded denied commands** (always blocked):

```
rm, rmdir, del, format, shutdown, reboot, halt, poweroff, dd, mkfs, fdisk
```

**Engine secrets stripped from environment** (never passed to child processes):

```
JWT_SECRET, DB_PASSWORD, ENCRYPTION_KEY, SMTP_PASSWORD,
STORAGE_S3_SECRET_KEY, STORAGE_S3_ACCESS_KEY
```

---

## Go API

### Enabling I/O Modules

```go
import (
    "github.com/anthropic/go-json"
    "github.com/anthropic/go-json/goio"
)

// Enable all I/O modules
rt := gojson.NewRuntime(gojson.WithIO(goio.All()))

// Enable specific modules only
rt := gojson.NewRuntime(gojson.WithIO(goio.HTTP(), goio.FS()))

// No I/O — default, safe for untrusted code
rt := gojson.NewRuntime(gojson.WithoutIO())
```

### Configuring Security

```go
rt := gojson.NewRuntime(
    gojson.WithIO(goio.All()),
    gojson.WithIOSecurity(&goio.SecurityConfig{
        FS: goio.FSSecurityConfig{
            AllowedPaths: []string{"/tmp/sandbox"},
            AllowWrite:   true,
            MaxFileSize:  10 * 1024 * 1024, // 10 MB
        },
        HTTP: goio.HTTPSecurityConfig{
            AllowedHosts:    []string{"api.example.com", "cdn.example.com"},
            MaxResponseSize: 5 * 1024 * 1024, // 5 MB
            Timeout:         30 * time.Second,
        },
        SQL: goio.SQLSecurityConfig{
            AllowedDrivers:  []string{"postgres", "sqlite"},
            MaxQueryTime:    10 * time.Second,
            MaxRows:         1000,
            BlockedKeywords: []string{"DROP", "ALTER", "TRUNCATE", "CREATE"},
        },
        Exec: goio.ExecSecurityConfig{
            AllowedCommands: []string{"ls", "cat", "grep", "wc", "node"},
            MaxTimeout:      30 * time.Second,
            MaxOutputSize:   512 * 1024, // 512 KB
        },
    }),
)
```

### Full Program Example

A go-json program that fetches data from an API, processes it, and writes the
result to a file:

**program.json:**

```json
{
  "import": {
    "http": "io:http",
    "fs": "io:fs"
  },
  "main": [
    {
      "let": "resp",
      "expr": "http.get('https://api.example.com/users')"
    },
    {
      "if": "resp.status != 200",
      "then": { "return": "{'error': 'API request failed', 'status': resp.status}" }
    },
    {
      "let": "activeUsers",
      "expr": "resp.body | filter(u => u.active)"
    },
    {
      "let": "report",
      "expr": "{'total': len(activeUsers), 'users': activeUsers | map(u => u.name)}"
    },
    {"let": "_", "expr": "fs.write('./report.json', toJSON(report))"},
    { "return": "report" }
  ]
}
```

**host.go:**

```go
program, err := gojson.CompileFile("program.json")
if err != nil {
    log.Fatal(err)
}

rt := gojson.NewRuntime(
    gojson.WithIO(goio.HTTP(), goio.FS()),
    gojson.WithIOSecurity(&goio.SecurityConfig{
        HTTP: goio.HTTPSecurityConfig{
            AllowedHosts: []string{"api.example.com"},
        },
        FS: goio.FSSecurityConfig{
            AllowedPaths: []string{"."},
            AllowWrite:   true,
        },
    }),
)

result, err := rt.Execute(program)
```
