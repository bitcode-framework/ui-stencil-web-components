# Phase 7: Module "setting" — Admin Panel as JSON Module

**Date**: 14 July 2026 (updated 16 July 2026)
**Status**: Draft
**Depends on**: Phase 4.5k (WASM + Native Plugins) + Phase 4.5h (Bridge Ergonomics) + all earlier phases (1-6C)
**Unlocks**: Production readiness, admin.go deprecation
**Master doc**: `2026-07-14-runtime-engine-redesign-master.md`

---

## Table of Contents

1. [Goal](#1-goal)
2. [Why This Phase Exists](#2-why-this-phase-exists)
3. [Architecture: Module "setting" Structure](#3-architecture-module-setting-structure)
4. [Models](#4-models)
5. [Views](#5-views)
6. [Processes](#6-processes)
7. [Scripts — Multi-Runtime Stress Test](#7-scripts--multi-runtime-stress-test)
8. [Security](#8-security)
9. [Migration from admin.go](#9-migration-from-admingo)
10. [Implementation Tasks](#10-implementation-tasks)

---

## 1. Goal

Replace the hardcoded `admin.go` (~2645 lines, 9 files) with a **JSON-defined module** called `setting`. This module uses the engine's own building blocks — models, views, processes, scripts — to provide the same admin functionality.

This is the **ultimate stress test**: if the engine can build its own admin panel as a module, it can build anything.

### 1.1 Success Criteria

- Module "setting" provides all functionality currently in `admin.go`
- **go-json is the primary runtime** — majority of processes written as go-json programs
- Uses **all 7 runtimes** in its scripts as proof of multi-runtime capability:
  - go-json (primary — business logic, CRUD, dashboard)
  - goja (embedded JS — lightweight validation)
  - yaegi (embedded Go — data export with goroutines)
  - Node.js/TS (external — complex validation with npm packages)
  - Python (external — system info, analytics)
  - WASM (embedded — compute-intensive task via wazero)
- Uses array-backed models for fixture/config data (Phase 6C)
- Uses metadata API for model/view introspection (Phase 6C)
- Uses fluent model API: `bc.model('user').where(...).count()` (Phase 4.5h)
- Uses imperative transactions: `bc.tx.begin/commit/rollback` (Phase 4.5h)
- Uses `script:` imports from go-json programs to call external scripts (Phase 4.5j)
- `admin.go` can be disabled via config (`admin.legacy = false`)
- Zero hardcoded HTML — all UI via views + templates

### 1.2 What This Phase Does NOT Do

- Does not delete `admin.go` — it becomes a fallback that can be disabled
- Does not change the engine core — only builds on top of it
- Does not add new engine features — all features should already exist from Phase 1-4.5k

### 1.3 Key Principle

> If module "setting" needs a feature that doesn't exist, that feature should be added to the appropriate earlier phase (1-4.5k), NOT hacked into module "setting".

This ensures the engine is genuinely capable, not just patched for one use case.

---

## 2. Why This Phase Exists

### 2.1 The Problem with admin.go

`admin.go` is **technical debt**:

1. **~2645 lines of hardcoded Go** — HTML strings, SQL queries, business logic all mixed
2. **Bypasses the engine** — doesn't use models, views, processes, or the bridge API
3. **Not extensible** — adding a new admin page requires Go code changes + recompile
4. **Not themeable** — hardcoded CSS inline
5. **Duplicates engine features** — has its own model listing, data tables, form rendering
6. **9 files** — admin.go, admin_api.go, admin_audit.go, admin_groups.go, admin_list_api.go, admin_models.go, admin_modules.go, admin_security.go, admin_views.go

### 2.2 The Solution

Build the same functionality as a **JSON module** that uses the engine's own capabilities:

```
admin.go (hardcoded):                    module "setting" (JSON-driven):
├── HTML string concatenation            ├── views/*.json (declarative)
├── Direct SQL queries                   ├── models/*.json (with array source)
├── Inline permission checks             ├── security/*.json (declarative)
├── Custom route handlers                ├── processes/*.json + *.go-json (go-json primary)
└── Hardcoded business logic             └── scripts/*.js/*.py/*.go/*.wasm (multi-runtime)
```

### 2.3 What admin.go Currently Does

| Feature | admin.go File | Module "setting" Equivalent |
|---------|--------------|---------------------------|
| Dashboard (stats) | admin.go | go-json process + custom view template |
| Model list | admin_models.go | go-json process via meta API + list view |
| Model detail (fields, indexes) | admin_models.go | go-json process + form view |
| Model data browser | admin_list_api.go | List view with dynamic model |
| Group management | admin_groups.go | CRUD views on `group` model |
| Group permissions | admin_groups.go | Form view with child tables |
| User management | admin.go | CRUD views on `user` model |
| Module list | admin_modules.go | go-json process via meta API + list view |
| View inspector | admin_views.go | go-json process + list/form views |
| Audit log | admin_audit.go | List view on `audit_log` model |
| Security history | admin_security.go | List view on `security_history` model |
| API explorer | admin_api.go | Custom view + meta API |
| Model JSON editor | admin_api.go | Form view with code editor widget |

---

## 3. Architecture: Module "setting" Structure

```
modules/setting/
├── module.json
├── models/
│   ├── _dashboard_stat.json          ← array model (process source)
│   ├── _nav_item.json                ← array model (admin navigation)
│   └── _system_info.json             ← array model (process source)
├── views/
│   ├── dashboard.json                ← custom view (dashboard)
│   ├── model_list.json               ← list view (all models)
│   ├── model_detail.json             ← form view (model inspector)
│   ├── model_data_list.json          ← list view (data browser)
│   ├── model_data_form.json          ← form view (record editor)
│   ├── group_list.json               ← list view
│   ├── group_form.json               ← form view
│   ├── user_list.json                ← list view
│   ├── user_form.json                ← form view
│   ├── module_list.json              ← list view
│   ├── audit_log_list.json           ← list view
│   ├── security_history_list.json    ← list view
│   └── api_explorer.json             ← custom view
├── processes/
│   ├── compute_dashboard_stats.json  ← go-json program (primary runtime)
│   ├── compute_system_info.json      ← go-json program with script: import (calls Python)
│   ├── list_models.json              ← go-json program (meta API)
│   ├── inspect_model.json            ← go-json program (meta API)
│   ├── impersonate_user.json         ← go-json program with bc.tx
│   ├── export_model_data.json        ← go-json program with script: import (calls yaegi)
│   ├── validate_model_schema.json    ← go-json program with script: import (calls Node.js)
│   └── compute_hash.json            ← go-json program with wasm: import (calls WASM)
├── scripts/
│   ├── system_info.py                ← Python (system introspection)
│   ├── data_export.go                ← Go/yaegi (CSV/XLSX with goroutines)
│   ├── model_validator.ts            ← TypeScript/Node.js (npm: ajv)
│   ├── quick_validate.js             ← JavaScript/goja (lightweight validation)
│   └── hash_compute.wasm             ← WASM (compute-intensive hashing)
├── security/
│   └── groups.json                   ← admin group permissions
└── templates/
    ├── dashboard.html                ← custom dashboard template
    └── api_explorer.html             ← API explorer template
```

### 3.1 module.json

```json
{
  "name": "setting",
  "label": "Settings & Administration",
  "version": "1.0.0",
  "depends": ["base"],
  "models": ["models/*.json"],
  "views": ["views/*.json"],
  "processes": ["processes/*.json"],
  "securities": ["security/*.json"],
  "table": { "prefix": "" }
}
```

No table prefix — setting module uses base tables directly (user, group, etc.) and its own array models.

---

## 4. Models

### 4.1 Array Models for Admin Data

Module "setting" uses **array-backed models** (Phase 6C) for data that doesn't need a database table:

#### Navigation Items

```json
{
  "name": "_setting_nav",
  "source": "array",
  "primary_key": { "strategy": "natural_key", "field": "key" },
  "fields": {
    "key": { "type": "string" },
    "label": { "type": "string" },
    "icon": { "type": "string" },
    "view": { "type": "string" },
    "group": { "type": "string" },
    "order": { "type": "integer" }
  },
  "rows": [
    { "key": "dashboard", "label": "Dashboard", "icon": "home", "view": "setting.dashboard", "group": "main", "order": 1 },
    { "key": "models", "label": "Models", "icon": "database", "view": "setting.model_list", "group": "schema", "order": 2 },
    { "key": "users", "label": "Users", "icon": "users", "view": "setting.user_list", "group": "security", "order": 3 },
    { "key": "groups", "label": "Groups", "icon": "shield", "view": "setting.group_list", "group": "security", "order": 4 },
    { "key": "modules", "label": "Modules", "icon": "package", "view": "setting.module_list", "group": "system", "order": 5 },
    { "key": "audit", "label": "Audit Log", "icon": "file-text", "view": "setting.audit_log_list", "group": "system", "order": 6 }
  ]
}
```

#### Dashboard Stats (Process Source)

```json
{
  "name": "_dashboard_stat",
  "source": "process",
  "process": "setting.compute_dashboard_stats",
  "refresh": "5m",
  "primary_key": { "strategy": "natural_key", "field": "key" },
  "fields": {
    "key": { "type": "string" },
    "label": { "type": "string" },
    "value": { "type": "string" },
    "icon": { "type": "string" },
    "color": { "type": "string" }
  }
}
```

### 4.2 Existing Base Models Used

Module "setting" does NOT create new tables for users, groups, etc. — it uses existing base models:

| Model | Source | Module |
|-------|--------|--------|
| `user` | base module | Existing DB model |
| `group` | base module | Existing DB model |
| `model_access` | base module | Existing DB model |
| `record_rule` | base module | Existing DB model |
| `audit_log` | base module | Existing DB model |
| `security_history` | base module | Existing DB model |

Module "setting" only adds **views and processes** for these models — no schema changes.

---

## 5. Views

### 5.1 Dashboard (Custom View)

```json
{
  "name": "dashboard",
  "type": "custom",
  "title": "Dashboard",
  "template": "templates/dashboard.html",
  "data_sources": {
    "stats": { "model": "_dashboard_stat" },
    "nav": { "model": "_setting_nav" },
    "recent_audit": {
      "model": "audit_log",
      "domain": [["action", "!=", "read"]],
      "limit": 10
    }
  }
}
```

### 5.2 Model List (Metadata)

```json
{
  "name": "model_list",
  "type": "list",
  "title": "Models",
  "data_sources": {
    "models": { "process": "setting.list_models" }
  },
  "fields": ["name", "module", "label", "field_count", "table_name"],
  "sort": { "field": "name", "order": "asc" },
  "actions": [
    { "label": "Inspect", "process": "setting.inspect_model", "variant": "primary" }
  ]
}
```

### 5.3 User Management

```json
{
  "name": "user_list",
  "type": "list",
  "model": "user",
  "title": "Users",
  "fields": ["name", "email", "last_login", "active"],
  "filters": ["name", "email", "active"],
  "actions": [
    { "label": "Impersonate", "process": "setting.impersonate_user", "permission": "setting.admin", "confirm": "Impersonate this user?" }
  ]
}
```

```json
{
  "name": "user_form",
  "type": "form",
  "model": "user",
  "title": "User",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "email", "width": 6 }
    ]},
    { "row": [
      { "field": "active", "width": 3 },
      { "field": "last_login", "width": 3, "readonly": true }
    ]},
    { "tabs": [
      { "label": "Groups", "view": "setting.user_groups", "filter_by": "user_id" },
      { "label": "Security History", "view": "setting.security_history_list", "filter_by": "user_id" },
      { "label": "Audit Log", "view": "setting.audit_log_list", "filter_by": "user_id" }
    ]}
  ]
}
```

### 5.4 Group Management with Permissions

```json
{
  "name": "group_form",
  "type": "form",
  "model": "group",
  "title": "Group",
  "layout": [
    { "row": [
      { "field": "name", "width": 6 },
      { "field": "display_name", "width": 6 }
    ]},
    { "tabs": [
      {
        "label": "Model Access",
        "view": "setting.model_access_list",
        "filter_by": "group_id"
      },
      {
        "label": "Record Rules",
        "view": "setting.record_rule_list",
        "filter_by": "group_id"
      },
      {
        "label": "Members",
        "view": "setting.group_members",
        "filter_by": "group_id"
      }
    ]}
  ]
}
```

---

## 6. Processes

### 6.1 Dashboard Stats — go-json (Primary Runtime)

```jsonc
{
  "name": "compute_dashboard_stats",
  "go_json": "1",
  "import": { "bc": "ext:bitcode" },
  "steps": [
    // Count models via meta API
    {"let": "models", "expr": "bc.call('_meta.list_models', {})"},
    {"let": "model_count", "expr": "len(models)"},

    // Count users and groups using fluent API (Phase 4.5h)
    {"let": "user_count", "expr": "bc.model('user').where('active', true).count()"},
    {"let": "group_count", "expr": "bc.model('group').count()"},

    // Count today's audit entries
    {"let": "today", "expr": "startOfDay(now())"},
    {"let": "audit_count", "expr": "bc.model('audit_log').where('created_at', '>=', today).count()"},

    // Return stats array for _dashboard_stat process source
    {"return": {"value": [
      {"key": "models", "label": "Models", "value": "string(model_count)", "icon": "database", "color": "#3b82f6"},
      {"key": "users", "label": "Active Users", "value": "string(user_count)", "icon": "users", "color": "#10b981"},
      {"key": "groups", "label": "Groups", "value": "string(group_count)", "icon": "shield", "color": "#f59e0b"},
      {"key": "audit_today", "label": "Actions Today", "value": "string(audit_count)", "icon": "activity", "color": "#8b5cf6"}
    ]}}
  ]
}
```

### 6.2 System Info — go-json with `script:` Import (Calls Python)

```jsonc
{
  "name": "compute_system_info",
  "go_json": "1",
  "import": {
    "bc": "ext:bitcode",
    "sysinfo": "script:./scripts/system_info.py"
  },
  "steps": [
    // Call Python script for OS-level info (Phase 4.5j script: import)
    {"let": "os_info", "call": "sysinfo.get_info"},

    // Get engine-level info via bridge
    {"let": "db_driver", "expr": "bc.config('db.driver') ?? 'sqlite'"},
    {"let": "module_list", "expr": "bc.call('_meta.list_modules', {})"},
    {"let": "module_count", "expr": "len(module_list)"},

    {"return": {"value": [
      {"key": "os", "label": "OS", "value": "os_info.os"},
      {"key": "db_driver", "label": "Database", "value": "db_driver"},
      {"key": "modules", "label": "Loaded Modules", "value": "string(module_count)"},
      {"key": "python", "label": "Python Version", "value": "os_info.python_version"}
    ]}}
  ]
}
```

### 6.3 Model Introspection — go-json (Primary Runtime)

```jsonc
{
  "name": "list_models",
  "go_json": "1",
  "import": { "bc": "ext:bitcode" },
  "steps": [
    {"let": "models", "expr": "bc.call('_meta.list_models', {})"},
    {"return": "models"}
  ]
}
```

```jsonc
{
  "name": "inspect_model",
  "go_json": "1",
  "import": { "bc": "ext:bitcode" },
  "input": { "model_name": "string" },
  "steps": [
    {"let": "detail", "expr": "bc.call('_meta.get_model', {'name': input.model_name})"},
    {"return": "detail"}
  ]
}
```

### 6.4 Impersonate User — go-json with Transaction (Phase 4.5h)

```jsonc
{
  "name": "impersonate_user",
  "go_json": "1",
  "import": { "bc": "ext:bitcode" },
  "input": { "target_user_id": "string" },
  "steps": [
    // Verify target user exists
    {"let": "target", "expr": "bc.model('user').find(input.target_user_id).get()"},
    {"if": "target == nil", "then": [
      {"error": "'User not found'"}
    ]},

    // Use transaction for audit + session swap (Phase 4.5h imperative tx)
    {"let": "_", "expr": "bc.tx.begin()"},
    {"try": [
      // Log impersonation in audit
      {"let": "_", "expr": "bc.audit.log({'action': 'impersonate', 'model': 'user', 'recordId': input.target_user_id, 'detail': 'Admin impersonated user ' + target.name})"},

      // Log in security history
      {"let": "_", "expr": "bc.model('security_history').create({'user_id': session.user_id, 'action': 'impersonate', 'target_user_id': input.target_user_id, 'ip': session.context.ip})"},

      {"let": "_", "expr": "bc.tx.commit()"}
    ], "catch": {
      "as": "err",
      "steps": [
        {"let": "_", "expr": "bc.tx.rollback()"},
        {"error": "err.message"}
      ]
    }},

    // Return impersonation token
    {"return": {"value": {"impersonated": true, "user_id": "input.target_user_id", "name": "target.name"}}}
  ]
}
```

### 6.5 Data Export — go-json with `script:` Import (Calls yaegi)

```jsonc
{
  "name": "export_model_data",
  "go_json": "1",
  "import": {
    "bc": "ext:bitcode",
    "exporter": "script:./scripts/data_export.go"
  },
  "input": {
    "model": "string",
    "format": "string"
  },
  "steps": [
    // Validate input
    {"if": "input.format not in ['json', 'csv', 'xlsx']", "then": [
      {"error": "'Invalid format. Must be json, csv, or xlsx'"}
    ]},

    // Fetch all records using fluent API
    {"let": "records", "expr": "bc.model(input.model).get()"},

    // Delegate to yaegi for file generation (goroutines for large datasets)
    {"let": "result", "call": "exporter.Export", "with": {
      "records": "records",
      "model": "input.model",
      "format": "input.format"
    }},

    {"return": "result"}
  ]
}
```

### 6.6 Model Schema Validation — go-json with `script:` Import (Calls Node.js)

```jsonc
{
  "name": "validate_model_schema",
  "go_json": "1",
  "import": {
    "bc": "ext:bitcode",
    "validator": "script:./scripts/model_validator.ts"
  },
  "input": { "model_json": "string" },
  "steps": [
    {"let": "result", "call": "validator.validate", "with": {
      "model_json": "input.model_json"
    }},
    {"if": "result.valid == false", "then": [
      {"return": {"value": {"valid": false, "errors": "result.errors"}}}
    ]},
    {"return": {"value": {"valid": true, "model": "result.model"}}}
  ]
}
```

### 6.7 Hash Compute — go-json with `wasm:` Import (Phase 4.5k)

```jsonc
{
  "name": "compute_hash",
  "go_json": "1",
  "import": {
    "hasher": "wasm:./scripts/hash_compute.wasm"
  },
  "input": { "data": "string", "algorithm": "string" },
  "steps": [
    // Call WASM module for compute-intensive hashing
    {"let": "result", "call": "hasher.hash", "with": {
      "data": "input.data",
      "algorithm": "input.algorithm"
    }},
    {"return": "result"}
  ]
}
```

---

## 7. Scripts — Multi-Runtime Stress Test

This is where module "setting" proves the engine's multi-runtime capability. **go-json is the primary runtime** for all processes. External scripts are called via `script:` and `wasm:` imports when a specific runtime's strength is needed.

### 7.1 Python — System Info (`scripts/system_info.py`)

```python
# Runtime: python (system introspection — platform module, os module)
# Called from: compute_system_info.json via script: import

import platform
import sys

def get_info():
    return {
        "os": platform.system() + " " + platform.release(),
        "python_version": sys.version.split()[0],
        "arch": platform.machine(),
        "hostname": platform.node(),
    }
```

### 7.2 Go/yaegi — Data Export (`scripts/data_export.go`)

```go
// Runtime: go (yaegi — goroutines for concurrent file generation)
// Called from: export_model_data.json via script: import

package main

import (
    "context"
    "encoding/csv"
    "encoding/json"
    "fmt"
    "strings"
)

func Execute(ctx context.Context, params map[string]any) (any, error) {
    records, _ := params["records"].([]any)
    modelName, _ := params["model"].(string)
    format, _ := params["format"].(string)

    switch format {
    case "csv":
        return exportCSV(records, modelName)
    case "json":
        data, err := json.MarshalIndent(records, "", "  ")
        if err != nil {
            return nil, err
        }
        return map[string]any{
            "filename": modelName + ".json",
            "content":  string(data),
            "mime":     "application/json",
        }, nil
    case "xlsx":
        return exportXLSX(records, modelName)
    default:
        return nil, fmt.Errorf("unsupported format: %s", format)
    }
}

func exportCSV(records []any, modelName string) (any, error) {
    if len(records) == 0 {
        return map[string]any{"filename": modelName + ".csv", "content": "", "mime": "text/csv"}, nil
    }

    var buf strings.Builder
    w := csv.NewWriter(&buf)

    // Header from first record keys
    first, _ := records[0].(map[string]any)
    var headers []string
    for k := range first {
        headers = append(headers, k)
    }
    w.Write(headers)

    // Rows
    for _, rec := range records {
        row, _ := rec.(map[string]any)
        var vals []string
        for _, h := range headers {
            vals = append(vals, fmt.Sprintf("%v", row[h]))
        }
        w.Write(vals)
    }
    w.Flush()

    return map[string]any{
        "filename": modelName + ".csv",
        "content":  buf.String(),
        "mime":     "text/csv",
    }, nil
}

func exportXLSX(records []any, modelName string) (any, error) {
    // yaegi can use excelize if available in go.mod
    // For now, fallback to CSV format with .xlsx extension note
    return nil, fmt.Errorf("XLSX export requires excelize package — use CSV for now")
}
```

### 7.3 TypeScript/Node.js — Model Validator (`scripts/model_validator.ts`)

```typescript
// Runtime: node (npm package: ajv for JSON Schema validation)
// Called from: validate_model_schema.json via script: import

import Ajv from "ajv";

const ajv = new Ajv();

export default {
  async execute(bitcode: any, params: any) {
    const modelJSON = params.model_json;
    const parsed = JSON.parse(modelJSON);

    // Load BitCode model schema from filesystem
    const schemaContent = await bitcode.fs.read("schemas/model.schema.json");
    const schema = JSON.parse(schemaContent);

    const validate = ajv.compile(schema);
    const valid = validate(parsed);

    if (!valid) {
      return { valid: false, errors: validate.errors };
    }

    return { valid: true, model: parsed };
  },
};
```

### 7.4 JavaScript/goja — Quick Validate (`scripts/quick_validate.js`)

```javascript
// Runtime: javascript (goja — embedded, fast, no npm needed)
// Called from go-json processes for lightweight validation

export default {
  execute(bitcode, params) {
    const data = params.data;
    const errors = [];

    if (!data.name || data.name.trim() === "") {
      errors.push("Name is required");
    }
    if (data.email && !data.email.includes("@")) {
      errors.push("Invalid email format");
    }

    return { valid: errors.length === 0, errors: errors };
  },
};
```

### 7.5 WASM — Hash Compute (`scripts/hash_compute.wasm`)

Pre-compiled WASM module for compute-intensive hashing. Source in Rust:

```rust
// scripts/build/hash_compute/src/lib.rs
// Compile: cargo build --target wasm32-wasi --release
// Output: scripts/hash_compute.wasm

use sha2::{Sha256, Sha512, Digest};
use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
struct HashInput {
    data: String,
    algorithm: String,
}

#[derive(Serialize)]
struct HashOutput {
    hash: String,
    algorithm: String,
    length: usize,
}

#[no_mangle]
pub extern "C" fn hash(ptr: *const u8, len: usize) -> u64 {
    let input_bytes = unsafe { std::slice::from_raw_parts(ptr, len) };
    let input: HashInput = serde_json::from_slice(input_bytes).unwrap();

    let hash_hex = match input.algorithm.as_str() {
        "sha256" => {
            let mut hasher = Sha256::new();
            hasher.update(input.data.as_bytes());
            format!("{:x}", hasher.finalize())
        }
        "sha512" => {
            let mut hasher = Sha512::new();
            hasher.update(input.data.as_bytes());
            format!("{:x}", hasher.finalize())
        }
        _ => return 0,
    };

    let output = HashOutput {
        length: hash_hex.len(),
        algorithm: input.algorithm,
        hash: hash_hex,
    };

    let json = serde_json::to_vec(&output).unwrap();
    let result_ptr = json.as_ptr() as u32;
    let result_len = json.len() as u32;
    std::mem::forget(json);
    ((result_ptr as u64) << 32) | (result_len as u64)
}

#[no_mangle]
pub extern "C" fn malloc(size: usize) -> *mut u8 {
    let mut buf = Vec::with_capacity(size);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

#[no_mangle]
pub extern "C" fn free(ptr: *mut u8, size: usize) {
    unsafe { Vec::from_raw_parts(ptr, 0, size); }
}
```

### 7.6 Runtime Distribution

| Script | Runtime | Called From | Why This Runtime |
|--------|---------|-----------|-----------------|
| (inline in .json) | **go-json** | All processes | Primary runtime — business logic, CRUD, bridge calls |
| system_info.py | Python | compute_system_info.json via `script:` | `platform` + `sys` modules for OS introspection |
| data_export.go | yaegi (Go) | export_model_data.json via `script:` | Goroutines for concurrent CSV/XLSX generation |
| model_validator.ts | Node.js | validate_model_schema.json via `script:` | npm package `ajv` for JSON Schema validation |
| quick_validate.js | goja (JS) | Direct from process step | Embedded, fast, no npm overhead |
| hash_compute.wasm | wazero (WASM) | compute_hash.json via `wasm:` | Compute-intensive, sandboxed, portable |

This proves **all 7 runtimes** work in a real module:
- **go-json**: Primary runtime for all process orchestration
- **goja**: Embedded JS for lightweight tasks
- **yaegi**: Embedded Go for concurrent/system tasks
- **Node.js/TS**: External JS for npm ecosystem
- **Python**: External for system introspection
- **WASM**: Embedded for compute-intensive sandboxed tasks

---

## 8. Security

### 8.1 Permission Group

```json
{
  "groups": {
    "setting.admin": {
      "label": "Settings Administrator",
      "permissions": [
        "setting.dashboard.read",
        "setting.model.read",
        "setting.model.write",
        "setting.user.read",
        "setting.user.write",
        "setting.group.read",
        "setting.group.write",
        "setting.audit.read",
        "setting.impersonate",
        "setting.export"
      ]
    },
    "setting.viewer": {
      "label": "Settings Viewer",
      "permissions": [
        "setting.dashboard.read",
        "setting.model.read",
        "setting.user.read",
        "setting.group.read",
        "setting.audit.read"
      ]
    }
  }
}
```

### 8.2 Access Control

All setting views and processes require `setting.*` permissions. Regular users cannot access admin functionality unless explicitly granted.

---

## 9. Migration from admin.go

### 9.1 Transition Strategy

```
Phase 1: Module "setting" built alongside admin.go
  → Both accessible: /admin (legacy) and /setting (new)
  → Feature parity validation

Phase 2: Module "setting" becomes default
  → /admin redirects to /setting
  → admin.go still available via config

Phase 3: admin.go deprecated
  → Config: admin.legacy = false (default)
  → admin.go code moved to archived/
  → Module "setting" is the only admin interface
```

### 9.2 Config

```toml
# bitcode.toml
[admin]
legacy = true       # true = admin.go active (default during transition)
                    # false = admin.go disabled, only module "setting"
```

### 9.3 Feature Parity Checklist

| Feature | admin.go | Module "setting" | Status |
|---------|:--------:|:----------------:|:------:|
| Dashboard with stats | ✅ | ⬜ | |
| Model list | ✅ | ⬜ | |
| Model field inspector | ✅ | ⬜ | |
| Model data browser | ✅ | ⬜ | |
| Model data CRUD | ✅ | ⬜ | |
| User list | ✅ | ⬜ | |
| User create/edit | ✅ | ⬜ | |
| Group list | ✅ | ⬜ | |
| Group permissions | ✅ | ⬜ | |
| Group members | ✅ | ⬜ | |
| Module list | ✅ | ⬜ | |
| View inspector | ✅ | ⬜ | |
| Audit log | ✅ | ⬜ | |
| Security history | ✅ | ⬜ | |
| User impersonation | ✅ | ⬜ | |
| API explorer | ✅ | ⬜ | |
| Model JSON editor | ✅ | ⬜ | |

---

## 10. Implementation Tasks

### 10.1 Module Structure

- [ ] Create `modules/setting/module.json`
- [ ] Create directory structure (models, views, processes, scripts, security, templates)
- [ ] Register module in embedded modules

### 10.2 Array Models

- [ ] Create `_setting_nav.json` (navigation items)
- [ ] Create `_dashboard_stat.json` (process source)
- [ ] Create `_system_info.json` (process source)

### 10.3 Views

- [ ] Dashboard (custom view + template)
- [ ] Model list
- [ ] Model detail/inspector
- [ ] Model data browser (dynamic model)
- [ ] Model data form (dynamic model)
- [ ] User list
- [ ] User form (with tabs: groups, security history, audit)
- [ ] Group list
- [ ] Group form (with tabs: model access, record rules, members)
- [ ] Module list
- [ ] Audit log list
- [ ] Security history list
- [ ] API explorer (custom view + template)

### 10.4 Processes (go-json Primary)

- [ ] compute_dashboard_stats.json — go-json with fluent model API
- [ ] compute_system_info.json — go-json with `script:` import (Python)
- [ ] list_models.json — go-json with meta API
- [ ] inspect_model.json — go-json with meta API
- [ ] impersonate_user.json — go-json with `bc.tx.begin/commit/rollback`
- [ ] export_model_data.json — go-json with `script:` import (yaegi)
- [ ] validate_model_schema.json — go-json with `script:` import (Node.js)
- [ ] compute_hash.json — go-json with `wasm:` import (WASM)

### 10.5 Scripts (Multi-Runtime)

- [ ] system_info.py — Python (system introspection)
- [ ] data_export.go — yaegi (CSV/XLSX with goroutines)
- [ ] model_validator.ts — Node.js/TS (npm: ajv)
- [ ] quick_validate.js — goja (lightweight validation)
- [ ] hash_compute.wasm — WASM (pre-compiled Rust, commit .wasm to repo)
- [ ] hash_compute Rust source + Makefile in `scripts/build/`

### 10.6 Security

- [ ] Create security groups (setting.admin, setting.viewer)
- [ ] Apply permissions to all views and processes

### 10.7 Templates

- [ ] Dashboard HTML template
- [ ] API explorer HTML template

### 10.8 Migration

- [ ] Add `admin.legacy` config
- [ ] Add redirect from /admin to /setting when legacy = false
- [ ] Feature parity testing against admin.go
- [ ] Documentation for migration

### 10.9 Testing

- [ ] Test all views render correctly
- [ ] Test all processes execute successfully
- [ ] Test go-json as primary runtime (dashboard stats, model introspection, impersonation)
- [ ] Test `script:` import works (Python, yaegi, Node.js)
- [ ] Test `wasm:` import works (hash compute)
- [ ] Test goja script works (quick validate)
- [ ] Test all 7 runtimes work in one module
- [ ] Test permissions (admin vs viewer vs unauthorized)
- [ ] Test array models load correctly
- [ ] Test process source models refresh correctly
- [ ] Test embedded view filter_by works in tabs
- [ ] Test transaction in impersonate_user (commit + rollback paths)
- [ ] Test fluent model API in dashboard stats
