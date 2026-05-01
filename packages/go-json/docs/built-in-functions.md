# Built-in Functions Reference

go-json provides **190+ built-in functions** across three layers:

| Layer | Source | Count | Description |
|-------|--------|-------|-------------|
| **Layer 1** | [expr-lang/expr](https://github.com/expr-lang/expr) | ~68 | Core expression functions — available in all expressions |
| **Layer 2** | go-json stdlib | 120+ | Additional functions registered by go-json |
| **Layer 3** | I/O modules + regex | varies | Side-effect functions (HTTP, FS, SQL, etc.) |

This document covers **Layer 1** and **Layer 2**. For Layer 3 (I/O), see [I/O Modules](io-modules.md).

---

## Function Namespacing

go-json functions use two calling styles: **flat** (no namespace) and **namespaced** (dot-prefixed). Understanding when each is used — and why — is important for writing idiomatic go-json programs.

### Flat Functions

Most functions are called directly by name:

```
upper("hello")                    // → "HELLO"
len(items)                        // → 5
"hello" contains "ell"            // → true (operator style)
strContains("hello", "ell")       // → true (function style)
abs(-42)                          // → 42
clamp(value, 0, 100)              // → bounded value
```

These are **general-purpose utilities** that every program might use. They come from two sources:
- **Layer 1** (expr-lang built-ins): `abs`, `ceil`, `len`, `upper`, `lower`, `trim`, `split`, `filter`, `map`, `sort`, etc.
- **Layer 2** (go-json stdlib): `clamp`, `pow`, `sqrt`, `padLeft`, `append`, `slice`, `formatDate`, `isNil`, etc.

From the programmer's perspective, there is no difference — both are called the same way.

### Namespaced Functions

Some functions are grouped under a **namespace prefix** accessed via dot notation:

```
crypto.sha256("hello")                    // → hash string
crypto.uuid()                             // → "550e8400-..."
regex.match("hello123", "\\d+")           // → true
regex.replace("hello", "[aeiou]", "*")    // → "h*ll*"
```

And I/O modules are always namespaced (via import alias):

```json
{"import": {"http": "io:http", "fs": "io:fs", "sql": "io:sql"}}
```
```
http.get("https://api.example.com/users")
fs.read("./config.json")
sql.query("SELECT * FROM users WHERE age > ?", [18])
```

### How Namespacing Works Internally

Namespaces are **not special syntax** — they use standard expr-lang member access on maps. A namespace is simply a `map[string]any` registered as a variable, where each value is a function:

```go
// This is how crypto.* is registered internally
r.RegisterEnv("crypto", map[string]any{
    "sha256": func(args ...any) (any, error) { ... },
    "md5":    func(args ...any) (any, error) { ... },
    "uuid":   func(args ...any) (any, error) { ... },
    "hmac":   func(args ...any) (any, error) { ... },
})
```

When you write `crypto.sha256("hello")`, expr-lang sees:
1. `crypto` → look up variable → finds a map
2. `.sha256` → member access on the map → finds a function
3. `("hello")` → call the function

This is **native expr-lang behavior**, not a go-json hack. Any map-of-functions works as a namespace.

### When to Use Which Style

| Style | When | Examples |
|-------|------|---------|
| **Flat** | General-purpose utility, no collision risk, everyone uses it | `len()`, `upper()`, `abs()`, `clamp()`, `strContains()` |
| **Namespaced** | Domain-specific, collision risk, or grouped by concern | `crypto.*`, `regex.*` |
| **Import-namespaced** | I/O modules, extensions, imported libraries | `http.*`, `fs.*`, `sql.*`, `ext:*` |

### Design Rationale

**Why `strContains()` has a `str` prefix:**

`contains`, `startsWith`, `endsWith`, and `matches` are **reserved operator keywords** in expr-lang. They work as infix operators (`"abc" contains "b"`) but cannot be called as functions (`contains("abc", "b")` → parse error). go-json provides function-call aliases with a `str` prefix: `strContains()`, `strStartsWith()`, `strEndsWith()`, `strMatches()`. Both styles work:

```
"hello" contains "ell"          // operator style (expr-lang built-in)
strContains("hello", "ell")     // function style (go-json stdlib)
```

**Why `crypto.*` is namespaced:**

`sha256` is domain-specific — it only makes sense in a cryptographic context. Namespacing it under `crypto.sha256()` provides:
- **Discoverability** — seeing `crypto.` tells you "this is crypto-related"
- **Collision avoidance** — `sha256` alone could conflict with a user variable
- **Grouping** — `crypto.sha256`, `crypto.md5`, `crypto.uuid`, `crypto.hmac` clearly belong together

**Why `regex.*` is namespaced:**

`regex.findAll()` and `regex.replace()` are specialized operations that return complex results. They're namespaced under `regex.*` to group them with `regex.match()` and signal "this is regex-specific".

Note that `"hello" matches "^h"` (operator, expr-lang) and `regex.match("hello", "^h")` (namespaced, go-json stdlib) and `strMatches("hello", "^h")` (flat function, go-json stdlib) all do the same thing — three ways to match a regex.

### The Complete Namespace Map

| Namespace | Functions | Source |
|-----------|-----------|--------|
| *(flat)* | ~160 functions | expr-lang built-ins + go-json stdlib |
| `crypto.*` | `sha256`, `sha512`, `md5`, `uuid`, `hmac`, `encrypt`, `decrypt`, `hashPassword`, `verifyPassword`, `randomBytes` | go-json stdlib |
| `validate.*` | `isEmail`, `isURL`, `isIP`, `isUUID`, `isJSON`, `isNumeric`, `isAlpha`, `isBase64`, `isHexColor`, `isCreditCard` | go-json stdlib |
| `regex.*` | `match`, `findAll`, `replace` | go-json stdlib |
| `http.*` | `get`, `post`, `put`, `patch`, `delete` | I/O module (import required) |
| `fs.*` | `read`, `write`, `append`, `exists`, `list`, `mkdir`, `remove`, `stat`, `copy`, `move`, `glob` | I/O module (import required) |
| `sql.*` | `query`, `execute`, `begin`, `commit`, `rollback` | I/O module (import required) |
| `exec.*` | `run` | I/O module (import required) |
| `mongo.*` | `find`, `findOne`, `insert`, `insertMany`, `update`, `delete`, `count`, `aggregate` | I/O module (import required) |
| `redis.*` | `get`, `set`, `del`, `exists`, `expire`, `ttl`, `incr`, `decr`, `hget`, `hset`, `hgetall`, `lpush`, `rpush`, `lrange`, `sadd`, `smembers`, `publish` | I/O module (import required) |
| `jwt.*` | `sign`, `verify`, `decode`, `refresh` | Server mode only |
| *custom* | any | Host extensions via `ext:name` |

### Creating Your Own Namespaces (Embedding)

When embedding go-json in a Go application, you can create custom namespaces using the extension API:

```go
rt := gojson.NewRuntime(
    gojson.WithExtension("myapp", runtime.Extension{
        Functions: map[string]any{
            // Flat function
            "getVersion": func() string { return "1.0.0" },
            // Namespaced functions
            "users": map[string]any{
                "find":   func(id string) (map[string]any, error) { ... },
                "create": func(data map[string]any) (map[string]any, error) { ... },
                "delete": func(id string) error { ... },
            },
            "cache": map[string]any{
                "get": func(key string) (any, error) { ... },
                "set": func(key string, val any, ttl int) error { ... },
            },
        },
    }),
)
```

In the program:

```json
{
  "import": {"app": "ext:myapp"},
  "steps": [
    {"let": "ver", "expr": "app.getVersion()"},
    {"let": "user", "expr": "app.users.find('user-123')"},
    {"let": "cached", "expr": "app.cache.get('key')"}
  ]
}
```

Nesting works to any depth — `app.db.pool.stats()` is valid as long as the map structure matches.

---

## Layer 1 — expr-lang Built-ins

These functions are provided by the [expr-lang/expr](https://github.com/expr-lang/expr) expression engine (v1.17+) and are available in every `expr` and `with` context. go-json does NOT reimplement these — they come for free. Full upstream docs: https://expr-lang.org/docs/language-definition

### Math (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `abs(x)` | `number → number` | Absolute value | `abs(-5)` → `5` |
| `ceil(x)` | `float → int` | Round up | `ceil(3.2)` → `4` |
| `floor(x)` | `float → int` | Round down | `floor(3.8)` → `3` |
| `round(x)` | `float → int` | Round to nearest | `round(3.5)` → `4` |
| `min(a, b, ...)` | `...number → number` | Minimum value | `min(3, 1, 4)` → `1` |
| `max(a, b, ...)` | `...number → number` | Maximum value | `max(3, 1, 4)` → `4` |
| `sum(arr)` | `[]number → number` | Sum of array | `sum([1, 2, 3])` → `6` |
| `mean(arr)` | `[]number → float` | Average of array | `mean([1, 2, 3])` → `2.0` |
| `median(arr)` | `[]number → float` | Median of array | `median([1, 2, 3])` → `2.0` |

### String (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `len(s)` | `string → int` | String length | `len("hello")` → `5` |
| `upper(s)` | `string → string` | Uppercase | `upper("hello")` → `"HELLO"` |
| `lower(s)` | `string → string` | Lowercase | `lower("HELLO")` → `"hello"` |
| `trim(s)` | `string → string` | Trim whitespace | `trim("  hi  ")` → `"hi"` |
| `trimPrefix(s, prefix)` | `string, string → string` | Remove prefix | `trimPrefix("hello", "he")` → `"llo"` |
| `trimSuffix(s, suffix)` | `string, string → string` | Remove suffix | `trimSuffix("hello", "lo")` → `"hel"` |
| `split(s, sep)` | `string, string → []string` | Split string | `split("a,b,c", ",")` → `["a","b","c"]` |
| `join(arr, sep)` | `[]string, string → string` | Join array | `join(["a","b"], ",")` → `"a,b"` |
| `replace(s, old, new)` | `string, string, string → string` | Replace all | `replace("hello", "l", "r")` → `"herro"` |
| `repeat(s, n)` | `string, int → string` | Repeat string | `repeat("ab", 3)` → `"ababab"` |
| `hasPrefix(s, prefix)` | `string, string → bool` | Starts with (function) | `hasPrefix("hello", "he")` → `true` |
| `hasSuffix(s, suffix)` | `string, string → bool` | Ends with (function) | `hasSuffix("hello", "lo")` → `true` |
| `indexOf(s, sub)` | `string, string → int` | First index (-1 if not found) | `indexOf("hello", "ll")` → `2` |
| `lastIndexOf(s, sub)` | `string, string → int` | Last index (-1 if not found) | `lastIndexOf("abcabc", "abc")` → `3` |
| `contains` | operator | Substring check | `"hello" contains "ell"` → `true`. Function alias: `strContains("hello", "ell")` |
| `startsWith` | operator | Prefix check | `"hello" startsWith "hel"` → `true`. Function alias: `strStartsWith("hello", "hel")` |
| `endsWith` | operator | Suffix check | `"hello" endsWith "llo"` → `true`. Function alias: `strEndsWith("hello", "llo")` |
| `matches` | operator | Regex match | `"hello" matches "^h"` → `true`. Also: `matches("hello", "^h")` (function alias, Layer 2) |

### Array (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `len(arr)` | `[]any → int` | Array length | `len([1,2,3])` → `3` |
| `first(arr)` | `[]any → any` | First element (nil if empty) | `first([1,2,3])` → `1` |
| `last(arr)` | `[]any → any` | Last element (nil if empty) | `last([1,2,3])` → `3` |
| `get(arr, index)` | `[]any, int → any` | Safe index access (nil if out of range) | `get([1,2,3], 0)` → `1` |
| `get(map, key)` | `map, string → any` | Safe key access (nil if missing) | `get(user, "name")` → `"Alice"` |
| `take(arr, n)` | `[]any, int → []any` | First n elements | `take([1,2,3,4], 2)` → `[1,2]` |
| `filter(arr, pred)` | `[]any, predicate → []any` | Filter elements | `filter(users, .age > 18)` |
| `map(arr, fn)` | `[]any, function → []any` | Transform elements | `map(users, .name)` |
| `reduce(arr, fn, init)` | `[]any, function, any → any` | Reduce to single value | `reduce([1,2,3], # + #acc, 0)` → `6` |
| `find(arr, pred)` | `[]any, predicate → any` | First matching element | `find(users, .name == 'Alice')` |
| `findIndex(arr, pred)` | `[]any, predicate → int` | Index of first match | `findIndex(users, .active)` |
| `findLast(arr, pred)` | `[]any, predicate → any` | Last matching element | `findLast(items, .price > 100)` |
| `findLastIndex(arr, pred)` | `[]any, predicate → int` | Index of last match | `findLastIndex(items, .active)` |
| `count(arr, pred)` | `[]any, predicate → int` | Count matching | `count(users, .active)` |
| `sum(arr, pred?)` | `[]any, predicate? → number` | Sum (with optional field) | `sum(orders, .total)` |
| `all(arr, pred)` | `[]any, predicate → bool` | All match predicate | `all(scores, # >= 60)` |
| `any(arr, pred)` | `[]any, predicate → bool` | Any match predicate | `any(users, .admin)` |
| `none(arr, pred)` | `[]any, predicate → bool` | None match predicate | `none(items, .deleted)` |
| `one(arr, pred)` | `[]any, predicate → bool` | Exactly one matches | `one(users, .winner)` |
| `sort(arr, order?)` | `[]any, string? → []any` | Sort (default asc) | `sort([3,1,2])` → `[1,2,3]` |
| `sortBy(arr, pred, order?)` | `[]any, predicate, string? → []any` | Sort by field | `sortBy(users, .age, "desc")` |
| `groupBy(arr, pred)` | `[]any, predicate → map` | Group by key | `groupBy(users, .department)` |
| `reverse(arr)` | `[]any → []any` | Reverse array | `reverse([1,2,3])` → `[3,2,1]` |
| `flatten(arr)` | `[][]any → []any` | Flatten one level | `flatten([[1,2],[3]])` → `[1,2,3]` |
| `uniq(arr)` | `[]any → []any` | Remove duplicates | `uniq([1,2,2,3])` → `[1,2,3]` |
| `concat(a, b, ...)` | `...[]any → []any` | Concatenate arrays | `concat([1,2], [3,4])` → `[1,2,3,4]` |

### Type Conversion & Serialization (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `int(x)` | `any → int` | Convert to integer | `int("42")` → `42` |
| `float(x)` | `any → float` | Convert to float | `float("3.14")` → `3.14` |
| `string(x)` | `any → string` | Convert to string | `string(42)` → `"42"` |
| `type(x)` | `any → string` | Get type name | `type(42)` → `"int"` |
| `toJSON(x)` | `any → string` | Serialize to JSON (pretty-printed) | `toJSON({"a": 1})` |
| `fromJSON(s)` | `string → any` | Parse JSON string | `fromJSON('{"a":1}')` → `{"a": 1}` |
| `toBase64(s)` | `string → string` | Base64 encode | `toBase64("hello")` → `"aGVsbG8="` |
| `fromBase64(s)` | `string → string` | Base64 decode | `fromBase64("aGVsbG8=")` → `"hello"` |
| `toPairs(m)` | `map → [][]any` | Map to key-value pairs | `toPairs({"a":1})` → `[["a",1]]` |
| `fromPairs(arr)` | `[][]any → map` | Key-value pairs to map | `fromPairs([["a",1]])` → `{"a":1}` |

### Date & Time (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `now()` | `→ datetime` | Current time | `now()` |
| `date(s, format?, tz?)` | `string, string?, string? → datetime` | Parse date string | `date("2024-01-15")` |
| `duration(s)` | `string → duration` | Parse duration (ns, us, ms, s, m, h) | `duration("2h30m")` |
| `timezone(s)` | `string → location` | Load timezone | `timezone("Asia/Jakarta")` |

Date objects expose methods: `.Year()`, `.Month()`, `.Day()`, `.Hour()`, `.Minute()`, `.Second()`, `.Weekday()`, `.YearDay()`, `.In(tz)`.

### Map/Object (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `len(m)` | `map → int` | Number of keys | `len({"a":1,"b":2})` → `2` |
| `keys(m)` | `map → []any` | Get all keys | `keys({"a":1,"b":2})` → `["a","b"]` |
| `values(m)` | `map → []any` | Get all values | `values({"a":1,"b":2})` → `[1,2]` |
| `get(m, key)` | `map, string → any` | Safe key access (nil if missing) | `get(user, "email")` → `nil` |

### Bitwise (expr-lang)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `bitand(a, b)` | `int, int → int` | Bitwise AND | `bitand(0b1010, 0b1100)` → `0b1000` |
| `bitor(a, b)` | `int, int → int` | Bitwise OR | `bitor(0b1010, 0b1100)` → `0b1110` |
| `bitxor(a, b)` | `int, int → int` | Bitwise XOR | `bitxor(0b1010, 0b1100)` → `0b0110` |
| `bitnot(a)` | `int → int` | Bitwise NOT | `bitnot(0b1010)` |
| `bitshl(a, n)` | `int, int → int` | Left shift | `bitshl(1, 4)` → `16` |
| `bitshr(a, n)` | `int, int → int` | Right shift | `bitshr(16, 4)` → `1` |

### Operators (expr-lang)

These are **infix operators**, not functions. They are used directly in expressions.

| Operator | Description | Example |
|----------|-------------|---------|
| `+`, `-`, `*`, `/`, `%` | Arithmetic | `a + b * 2` |
| `**` or `^` | Exponentiation | `2 ** 10` → `1024` |
| `==`, `!=`, `>`, `<`, `>=`, `<=` | Comparison | `age >= 18` |
| `and` / `&&`, `or` / `||`, `not` / `!` | Logical | `active && !deleted` |
| `? :` | Ternary | `age >= 18 ? 'adult' : 'minor'` |
| `if {} else {}` | Multiline conditional | `if (x > 0) { x } else { -x }` |
| `??` | Nil coalescing | `name ?? 'Anonymous'` |
| `?.` | Optional chaining | `user?.address?.city` |
| `\|` | Pipe | `items \| filter(.active) \| map(.name)` |
| `in`, `not in` | Membership | `'admin' in user.roles` |
| `matches`, `not matches` | Regex match (operator) | `email matches '^[a-z]+@'` |
| `contains`, `not contains` | String/array contains (operator) | `name contains 'alice'` |
| `startsWith`, `not startsWith` | String prefix (operator) | `name startsWith 'A'` |
| `endsWith`, `not endsWith` | String suffix (operator) | `name endsWith 'son'` |
| `..` | Range | `1..10` → `[1,2,...,10]` |
| `[:]` | Slice | `arr[1:3]`, `arr[:2]`, `arr[2:]` |
| `let` | Variable declaration | `let x = 42; x * 2` |

### Predicate Syntax (expr-lang)

In array functions, use `#` for the current element, `#acc` for the accumulator, and `#index` for the index:

```
filter(items, # > 3)              // items where value > 3
filter(users, .age > 18)          // users where .age > 18 (shorthand for #.age)
map(users, .name)                 // extract name from each user
reduce(items, # + #acc, 0)        // sum all items
all(scores, # >= 60)              // all scores >= 60
count(users, .active)             // count active users
sum(orders, .total)               // sum order totals
```

---

## Layer 2 — go-json Stdlib

These functions are added by go-json on top of expr-lang. They are registered via the `stdlib` package.

### Math (23 functions + 4 constants)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `clamp(x, min, max)` | `number, number, number → number` | Clamp value to range | `clamp(15, 0, 10)` → `10` |
| `sign(x)` | `number → int` | Sign of number (-1, 0, 1) | `sign(-5)` → `-1` |
| `pow(base, exp)` | `number, number → float` | Power | `pow(2, 10)` → `1024.0` |
| `sqrt(x)` | `number → float` | Square root | `sqrt(144)` → `12.0` |
| `mod(a, b)` | `number, number → number` | Modulo (always positive) | `mod(17, 5)` → `2` |
| `randomInt(min, max)` | `int, int → int` | Random integer in range [min, max) | `randomInt(1, 100)` |
| `randomFloat(min, max)` | `float, float → float` | Random float in range [min, max) | `randomFloat(0.0, 1.0)` |
| `sin(x)` | `float → float` | Sine (radians) | `sin(PI / 2)` → `1.0` |
| `cos(x)` | `float → float` | Cosine (radians) | `cos(0)` → `1.0` |
| `tan(x)` | `float → float` | Tangent (radians) | `tan(PI / 4)` → `1.0` |
| `asin(x)` | `float → float` | Arc sine | `asin(1)` → `PI/2` |
| `acos(x)` | `float → float` | Arc cosine | `acos(1)` → `0` |
| `atan(x)` | `float → float` | Arc tangent | `atan(1)` → `PI/4` |
| `atan2(y, x)` | `float, float → float` | Two-argument arc tangent | `atan2(1, 1)` → `PI/4` |
| `log(x)` | `float → float` | Natural logarithm | `log(E)` → `1.0` |
| `log2(x)` | `float → float` | Base-2 logarithm | `log2(8)` → `3.0` |
| `log10(x)` | `float → float` | Base-10 logarithm | `log10(1000)` → `3.0` |
| `exp(x)` | `float → float` | e^x | `exp(1)` → `E` |
| `trunc(x)` | `float → float` | Truncate toward zero | `trunc(3.7)` → `3.0` |
| `random()` | `→ float` | Random float [0, 1) | `random()` → `0.42...` |
| `isNaN(x)` | `any → bool` | Check if NaN | `isNaN(0.0/0.0)` → `true` |
| `isInfinite(x)` | `any → bool` | Check if infinite | `isInfinite(1.0/0.0)` → `true` |
| `isFinite(x)` | `any → bool` | Not NaN and not Inf | `isFinite(42)` → `true` |

**Constants** (available as variables in all expressions):

| Constant | Value | Description |
|----------|-------|-------------|
| `PI` | `3.141592653589793` | Pi |
| `E` | `2.718281828459045` | Euler's number |
| `Infinity` | `+Inf` | Positive infinity |
| `NaN` | `NaN` | Not a Number |

### String (28 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `padLeft(s, length, char)` | `string, int, string → string` | Left-pad string | `padLeft("42", 5, "0")` → `"00042"` |
| `padRight(s, length, char)` | `string, int, string → string` | Right-pad string | `padRight("hi", 5, ".")` → `"hi..."` |
| `substring(s, start, end?)` | `string, int, int? → string` | Extract substring | `substring("hello", 1, 3)` → `"el"` |
| `format(template, args...)` | `string, ...any → string` | Template formatting | `format("Hello, %s!", "Alice")` → `"Hello, Alice!"` |
| `strMatches(s, pattern)` | `string, string → bool` | Regex match (function alias for `matches` operator) | `strMatches("hello123", "^[a-z]+\\d+$")` → `true` |
| `strContains(s, substr)` | `string, string → bool` | Substring check (function alias for `contains` operator) | `strContains("hello world", "world")` → `true` |
| `strStartsWith(s, prefix)` | `string, string → bool` | Prefix check (function alias for `startsWith` operator) | `strStartsWith("hello", "hel")` → `true` |
| `strEndsWith(s, suffix)` | `string, string → bool` | Suffix check (function alias for `endsWith` operator) | `strEndsWith("hello", "llo")` → `true` |
| `capitalize(s)` | `string → string` | First char uppercase | `capitalize("hello")` → `"Hello"` |
| `title(s)` | `string → string` | Title case (each word) | `title("hello world")` → `"Hello World"` |
| `camelCase(s)` | `string → string` | Convert to camelCase | `camelCase("hello_world")` → `"helloWorld"` |
| `snakeCase(s)` | `string → string` | Convert to snake_case | `snakeCase("helloWorld")` → `"hello_world"` |
| `kebabCase(s)` | `string → string` | Convert to kebab-case | `kebabCase("helloWorld")` → `"hello-world"` |
| `pascalCase(s)` | `string → string` | Convert to PascalCase | `pascalCase("hello_world")` → `"HelloWorld"` |
| `truncate(s, max, suffix?)` | `string, int, string? → string` | Truncate with suffix (default "...") | `truncate("hello world", 8)` → `"hello..."` |
| `slugify(s)` | `string → string` | URL-friendly slug | `slugify("Hello World!")` → `"hello-world"` |
| `strReverse(s)` | `string → string` | Reverse string (rune-aware) | `strReverse("hello")` → `"olleh"` |
| `strCount(s, sub)` | `string, string → int` | Count substring occurrences | `strCount("hello", "l")` → `2` |
| `replaceFirst(s, old, new)` | `string, string, string → string` | Replace first occurrence | `replaceFirst("aaa", "a", "b")` → `"baa"` |
| `lines(s)` | `string → []string` | Split by newlines | `lines("a\nb\nc")` → `["a","b","c"]` |
| `words(s)` | `string → []string` | Split by whitespace | `words("hello  world")` → `["hello","world"]` |
| `isDigit(s)` | `string → bool` | All chars are digits | `isDigit("123")` → `true` |
| `isAlpha(s)` | `string → bool` | All chars are letters | `isAlpha("hello")` → `true` |
| `isAlphaNum(s)` | `string → bool` | All chars are letters or digits | `isAlphaNum("abc123")` → `true` |
| `isEmpty(s)` | `string → bool` | String is "" | `isEmpty("")` → `true` |
| `isBlank(s)` | `string → bool` | Empty or whitespace only | `isBlank("  ")` → `true` |
| `escapeHTML(s)` | `string → string` | HTML entity encoding | `escapeHTML("<b>hi</b>")` → `"&lt;b&gt;hi&lt;/b&gt;"` |
| `unescapeHTML(s)` | `string → string` | HTML entity decoding | `unescapeHTML("&lt;b&gt;")` → `"<b>"` |

### Array (19 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `append(arr, item)` | `[]any, any → []any` | Append to array | `append([1,2], 3)` → `[1,2,3]` |
| `prepend(arr, item)` | `[]any, any → []any` | Prepend to array | `prepend([2,3], 1)` → `[1,2,3]` |
| `slice(arr, start, end?)` | `[]any, int, int? → []any` | Slice array | `slice([1,2,3,4], 1, 3)` → `[2,3]` |
| `chunk(arr, size)` | `[]any, int → [][]any` | Split into chunks | `chunk([1,2,3,4,5], 2)` → `[[1,2],[3,4],[5]]` |
| `zip(a, b)` | `[]any, []any → [][]any` | Zip two arrays | `zip([1,2], ["a","b"])` → `[[1,"a"],[2,"b"]]` |
| `compact(arr)` | `[]any → []any` | Remove nil values | `compact([1, nil, 2, nil])` → `[1, 2]` |
| `includes(arr, item)` | `[]any, any → bool` | Check if array contains item | `includes([1,2,3], 2)` → `true` |
| `arrayIndexOf(arr, item)` | `[]any, any → int` | Find index (-1 if not found) | `arrayIndexOf(["a","b","c"], "b")` → `1` |
| `keyBy(arr, key)` | `[]map, string → map` | Array of maps → map keyed by field | `keyBy(users, "id")` |
| `difference(a, b)` | `[]any, []any → []any` | Elements in a not in b | `difference([1,2,3], [2])` → `[1,3]` |
| `intersection(a, b)` | `[]any, []any → []any` | Elements in both | `intersection([1,2,3], [2,3,4])` → `[2,3]` |
| `union(a, b)` | `[]any, []any → []any` | Unique elements from both | `union([1,2], [2,3])` → `[1,2,3]` |
| `fill(n, value)` | `int, any → []any` | Create array of N identical values | `fill(3, 0)` → `[0,0,0]` |
| `drop(arr, n)` | `[]any, int → []any` | Remove first N elements | `drop([1,2,3,4], 2)` → `[3,4]` |
| `takeRight(arr, n)` | `[]any, int → []any` | Last N elements | `takeRight([1,2,3,4], 2)` → `[3,4]` |
| `flatMap(arr, field)` | `[]map, string → []any` | Extract array field from each item, flatten | `flatMap(users, "tags")` → all tags |
| `partition(arr, field)` | `[]map, string → [[]any, []any]` | Split by truthy/falsy field value | `partition(users, "active")` → `[[active], [inactive]]` |

### Map/Object (13 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `has(obj, key)` | `map, string → bool` | Check if key exists | `has(config, "debug")` → `true` |
| `getIn(obj, path, sep?)` | `any, string, string? → any` | Deep nested traversal (nil-safe). Supports dot path and array index. Default separator `"."` | `getIn(user, "address.city")` → `"Jakarta"`, `getIn(data, "users[0].name")` → `"Alice"` |
| `merge(a, b)` | `map, map → map` | Shallow merge (b overrides a) | `merge(defaults, overrides)` |
| `pick(obj, keys)` | `map, []string → map` | Pick subset of keys | `pick(user, ["name", "email"])` |
| `omit(obj, keys)` | `map, []string → map` | Remove specified keys | `omit(user, ["password", "secret"])` |
| `deepMerge(a, b)` | `map, map → map` | Recursive merge (b overrides a) | `deepMerge(base, overrides)` |
| `deepClone(obj)` | `any → any` | Deep copy via JSON round-trip | `deepClone(original)` |
| `deepEqual(a, b)` | `any, any → bool` | Deep equality check | `deepEqual(obj1, obj2)` |
| `setIn(obj, path, value)` | `map, string, any → map` | Set nested value by dot-path (immutable) | `setIn(config, "db.host", "localhost")` |
| `deleteIn(obj, path)` | `map, string → map` | Delete nested key by dot-path (immutable) | `deleteIn(user, "password")` |
| `defaults(obj, defaults)` | `map, map → map` | Fill missing keys from defaults | `defaults(config, {"port": 8080})` |
| `mapKeys(obj, transform)` | `map, string → map` | Transform all keys (camelCase, snakeCase, kebabCase, pascalCase, upper, lower) | `mapKeys(obj, "camelCase")` |
| `mapValues(obj, transform)` | `map, string → map` | Transform all values (upper, lower, trim, string, int, float) | `mapValues(obj, "trim")` |

### DateTime (15 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `formatDate(dt, format)` | `datetime, string → string` | Format datetime | `formatDate(now(), "YYYY-MM-DD")` |
| `addDuration(dt, dur)` | `datetime, string → datetime` | Add duration to datetime | `addDuration(now(), "7d")` |
| `diffDates(a, b)` | `datetime, datetime → duration` | Difference between dates | `diffDates(end, start)` |
| `toUnix(dt)` | `datetime → int` | Unix timestamp (seconds) | `toUnix(now())` → `1720000000` |
| `fromUnix(ts)` | `int → datetime` | Unix timestamp to datetime | `fromUnix(1720000000)` |
| `toISO(dt)` | `datetime → string` | ISO 8601 string | `toISO(now())` → `"2024-07-03T..."` |
| `startOfDay(dt)` | `datetime → datetime` | 00:00:00 of the day | `startOfDay(now())` |
| `endOfDay(dt)` | `datetime → datetime` | 23:59:59 of the day | `endOfDay(now())` |
| `startOfMonth(dt)` | `datetime → datetime` | First day of month 00:00 | `startOfMonth(now())` |
| `endOfMonth(dt)` | `datetime → datetime` | Last day of month 23:59:59 | `endOfMonth(now())` |
| `isWeekend(dt)` | `datetime → bool` | Saturday or Sunday | `isWeekend(now())` |
| `isBefore(a, b)` | `datetime, datetime → bool` | a < b | `isBefore(start, end)` |
| `isAfter(a, b)` | `datetime, datetime → bool` | a > b | `isAfter(end, start)` |
| `daysInMonth(dt)` | `datetime → int` | Days in the month (28-31) | `daysInMonth(date("2024-02-01"))` → `29` |
| `isLeapYear(dt)` | `datetime → bool` | Leap year check | `isLeapYear(date("2024-01-01"))` → `true` |

`formatDate` supports **universal format tokens** that are auto-translated to Go layout:

| Token | Meaning | Example |
|-------|---------|---------|
| `YYYY` | 4-digit year | `2024` |
| `MM` | 2-digit month | `01` |
| `DD` | 2-digit day | `15` |
| `HH` | 2-digit hour (24h) | `14` |
| `mm` | 2-digit minute | `30` |
| `ss` | 2-digit second | `05` |

### Encoding (2 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `urlEncode(s)` | `string → string` | URL-encode string | `urlEncode("hello world")` → `"hello+world"` |
| `urlDecode(s)` | `string → string` | URL-decode string | `urlDecode("hello+world")` → `"hello world"` |

### Format (5 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `sprintf(fmt, args...)` | `string, ...any → string` | Printf-style formatting | `sprintf("%.2f", 3.14159)` → `"3.14"` |
| `toFixed(x, decimals?)` | `float, int? → string` | Format to N decimal places (default 2) | `toFixed(3.14159, 2)` → `"3.14"` |
| `formatNumber(n, decimals?, sep?, decSep?)` | `float, ... → string` | Number with thousand separators | `formatNumber(1234567.89, 2)` → `"1,234,567.89"` |
| `formatBytes(n)` | `int → string` | Human-readable bytes | `formatBytes(1048576)` → `"1.00 MB"` |
| `formatPercent(n, decimals?)` | `float, int? → string` | Format as percentage | `formatPercent(0.156, 1)` → `"15.6%"` |

### Type (2 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `bool(x)` | `any → bool` | Truthiness coercion | `bool(0)` → `false`, `bool("hi")` → `true` |
| `isNil(x)` | `any → bool` | Check if nil | `isNil(null)` → `true` |

### Environment (1 function)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `env(key, default?)` | `string, string? → string` | Read environment variable with optional default | `env("PORT", "8080")` |

The `env()` function supports access control via `EnvAccessConfig` (allow/deny glob patterns). When embedding go-json, use `WithEnvResolver` to customize the resolver and `WithEnvAccess` to restrict which variables are accessible.

### Crypto (11 functions — namespaced)

Crypto functions are accessed via the `crypto.` namespace:

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `crypto.sha256(s)` | `string → string` | SHA-256 hash (hex) | `crypto.sha256("hello")` |
| `crypto.sha512(s)` | `string → string` | SHA-512 hash (hex) | `crypto.sha512("hello")` |
| `crypto.md5(s)` | `string → string` | MD5 hash (hex) | `crypto.md5("hello")` |
| `crypto.uuid()` | `→ string` | Generate UUID v4 | `crypto.uuid()` → `"550e8400-..."` |
| `crypto.hmac(s, key, algo?)` | `string, string, string? → string` | HMAC signature | `crypto.hmac("data", "secret")` |
| `crypto.encrypt(plaintext, key)` | `string, string → string` | AES-256-GCM encrypt → base64 | `crypto.encrypt("secret", "mykey")` |
| `crypto.decrypt(ciphertext, key)` | `string, string → string` | AES-256-GCM decrypt from base64 | `crypto.decrypt(encrypted, "mykey")` |
| `crypto.hashPassword(password)` | `string → string` | bcrypt hash (cost 10) | `crypto.hashPassword("pass123")` |
| `crypto.verifyPassword(password, hash)` | `string, string → bool` | bcrypt verify | `crypto.verifyPassword("pass123", hash)` |
| `crypto.randomBytes(n)` | `int → string` | Crypto-secure random bytes (hex) | `crypto.randomBytes(16)` → 32-char hex |

`crypto.hmac` defaults to SHA-256. Pass `"sha512"` as the third argument for SHA-512.
`crypto.encrypt`/`decrypt` use AES-256-GCM. Key is normalized to 32 bytes via SHA-256 hash.

### Validate (10 functions — namespaced)

Validation functions are accessed via the `validate.` namespace:

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `validate.isEmail(s)` | `string → bool` | RFC 5322 basic email check | `validate.isEmail("user@example.com")` → `true` |
| `validate.isURL(s)` | `string → bool` | Valid URL with scheme | `validate.isURL("https://example.com")` → `true` |
| `validate.isIP(s)` | `string → bool` | IPv4 or IPv6 address | `validate.isIP("192.168.1.1")` → `true` |
| `validate.isUUID(s)` | `string → bool` | UUID v4 format | `validate.isUUID("550e8400-e29b-41d4-a716-446655440000")` → `true` |
| `validate.isJSON(s)` | `string → bool` | Valid JSON string | `validate.isJSON('{"a":1}')` → `true` |
| `validate.isNumeric(s)` | `string → bool` | Numeric string | `validate.isNumeric("3.14")` → `true` |
| `validate.isAlpha(s)` | `string → bool` | Letters only | `validate.isAlpha("hello")` → `true` |
| `validate.isBase64(s)` | `string → bool` | Valid base64 | `validate.isBase64("aGVsbG8=")` → `true` |
| `validate.isHexColor(s)` | `string → bool` | #RGB or #RRGGBB | `validate.isHexColor("#FF0000")` → `true` |
| `validate.isCreditCard(s)` | `string → bool` | Luhn algorithm check | `validate.isCreditCard("4111111111111111")` → `true` |

### Regex (3 functions — namespaced)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `regex.match(s, pattern)` | `string, string → bool` | Test if string matches pattern | `regex.match("hello123", "\\d+")` → `true` |
| `regex.findAll(s, pattern)` | `string, string → []string` | Find all matches | `regex.findAll("a1b2c3", "\\d")` → `["1","2","3"]` |
| `regex.replace(s, pattern, repl)` | `string, string, string → string` | Replace matches | `regex.replace("hello", "[aeiou]", "*")` → `"h*ll*"` |

Compiled regexes are cached (LRU, max 1000 patterns). ReDoS prevention: max pattern length 1000 chars, max input 1MB.

### Path (8 functions)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `basename(path)` | `string → string` | File name from path | `basename("/home/user/file.txt")` → `"file.txt"` |
| `dirname(path)` | `string → string` | Directory from path | `dirname("/home/user/file.txt")` → `"/home/user"` |
| `extname(path)` | `string → string` | File extension | `extname("file.txt")` → `".txt"` |
| `stemname(path)` | `string → string` | File name without extension | `stemname("file.txt")` → `"file"` |
| `joinpath(parts...)` | `...string → string` | Join path segments | `joinpath("home", "user", "file.txt")` |
| `cleanpath(path)` | `string → string` | Clean path (resolve `.`, `..`) | `cleanpath("/a/b/../c")` → `"/a/c"` |
| `isabs(path)` | `string → bool` | Is absolute path | `isabs("/home")` → `true` |
| `pathsep()` | `→ string` | OS path separator | `pathsep()` → `"/"` or `"\\"` |

### JSON (1 function)

| Function | Signature | Description | Example |
|----------|-----------|-------------|---------|
| `toCompactJSON(value)` | `any → string` | Serialize to compact JSON string (no indentation) | `toCompactJSON({"a": 1})` → `'{"a":1}'` |

`toJSON` and `fromJSON` are provided by expr-lang as built-in functions. `toJSON` produces pretty-printed output; use `toCompactJSON` when compact output is needed.

---

## Function Usage in Programs

### In Expressions (`expr`)

```json
{"let": "result", "expr": "clamp(input.value, 0, 100)"}
{"let": "hash", "expr": "crypto.sha256(input.password)"}
{"let": "names", "expr": "map(users, .name) | sort()"}
```

### In Computed Objects (`with`)

```json
{"let": "profile", "with": {
  "name": "upper(input.name)",
  "age_group": "input.age >= 18 ? 'adult' : 'minor'",
  "id": "crypto.uuid()"
}}
```

### In Conditions

```json
{"if": "len(items) > 0 && all(items, .valid)", "then": [...]}
{"while": "count < max(limits)", "steps": [...]}
```

### Pipe Chaining

```json
{"let": "active_names", "expr": "users | filter(.active) | map(.name) | sort()"}
{"let": "summary", "expr": "orders | filter(.total > 100) | map(.total) | sum()"}
```
