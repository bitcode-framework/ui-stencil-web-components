# @bitcode/tauri

Tauri 2.0 native shell for BitCode. Wraps @bitcode/components Stencil Web Components into desktop and mobile apps with offline SQLite storage, native capabilities, and offline-first sync.

## What is @bitcode/tauri?

A thin Rust shell around the BitCode Stencil component library. Tauri renders the same web components that run in the browser, but adds native capabilities: local SQLite with auto-migrations, file system access, push notifications, barcode scanning, biometric auth, and encrypted storage.

The app ships as a native binary on five platforms. The `bc-native.ts` bridge abstracts native calls behind a clean API, with Web API fallback when running outside Tauri. The `offline-store.ts` intercept layer routes CRUD to local SQLite for offline models and to `fetch()` for online models, so existing Stencil components work without modification.

**Platform support:**
- Desktop: Windows (.msi), macOS (.dmg), Linux (.deb, .AppImage)
- Mobile: iOS (.ipa), Android (.apk, .aab)

## Features

- **Five platforms** from one codebase: Windows, macOS, Linux, iOS, Android
- **Offline SQLite** with 6 auto-migrated tables (outbox, sync state, conflict log, number sequences, auth cache, model registry)
- **Native bridge** (bc-native): 13 methods covering database, filesystem, HTTP, notifications, barcode scanning, biometric auth, device info, network status, and badge counts
- **Optional SQLite encryption** via SQLCipher (feature flag: `encryption`)
- **Mobile plugins** behind feature flag (`mobile-plugins`): barcode scanner, biometric authentication
- **File system access** and **push notifications** via Tauri plugins
- **withGlobalTauri** mode: Stencil components stay framework-agnostic, no npm dependency on @tauri-apps/api

## Prerequisites

- [Rust](https://rustup.rs/) (latest stable)
- [Tauri CLI prerequisites](https://v2.tauri.app/start/prerequisites/) for your platform
- Node.js 18+ (for building Stencil components)
- For Android: [Android Studio](https://developer.android.com/studio) with NDK
- For iOS: [Xcode](https://developer.apple.com/xcode/) (macOS only)

## Quick Start

### Desktop

```bash
# Install component dependencies and build
cd packages/components && npm install

# Launch desktop dev window (1280x800, resizable, centered)
cd ../tauri && npm run dev:desktop
```

Tauri builds the Stencil components first (`beforeDevCommand`), then opens a native window pointing at `components/www`.

### Mobile

```bash
cd packages/tauri

# Android (first time)
npm run build:android-init    # Generate Android project
npm run dev:android           # Dev mode on device/emulator

# iOS (first time, macOS only)
npm run build:ios-init        # Generate Xcode project
npm run dev:ios               # Dev mode on simulator/device
```

## Build Commands

| Script | Command | Description |
|--------|---------|-------------|
| `dev` | `cargo tauri dev` | Desktop dev server |
| `dev:desktop` | `cargo tauri dev` | Desktop dev server (alias) |
| `dev:android` | `cargo tauri android dev` | Android dev on device/emulator |
| `dev:ios` | `cargo tauri ios dev` | iOS dev on simulator/device |
| `build` | `cargo tauri build` | Desktop release build |
| `build:desktop` | `cargo tauri build` | Desktop release build (alias) |
| `build:desktop:debug` | `cargo tauri build --debug` | Desktop debug build (faster, no optimization) |
| `build:android-init` | `cargo tauri android init` | Generate Android project (first time) |
| `build:android` | `cargo tauri android build` | Android release APK/AAB |
| `build:android:debug` | `cargo tauri android build --debug` | Android debug build |
| `build:ios-init` | `cargo tauri ios init` | Generate Xcode project (first time) |
| `build:ios` | `cargo tauri ios build` | iOS release IPA |
| `build:ios:debug` | `cargo tauri ios build --debug` | iOS debug build |
| `icons` | `cargo tauri icon src-tauri/icons/app-icon.png -o src-tauri/icons` | Generate all platform icons from source |
| `check` | `cd src-tauri && cargo check` | Rust type check without full build |

Release output goes to `src-tauri/target/release/bundle/`.

## Offline Schema

Tauri creates 6 SQLite tables automatically on first launch via `tauri-plugin-sql` migrations. These are infrastructure tables, not user-defined models.

| Table | Purpose |
|-------|---------|
| `_off_outbox` | Pending sync operations (CREATE/UPDATE/DELETE) with idempotency keys, retry counts, envelope grouping |
| `_off_sync_state` | Per-device sync state: last sync timestamp, pull version, registration info, auth cache timing |
| `_off_conflict_log` | Field-level conflict records with local/remote/resolved values and resolution strategy |
| `_off_number_sequence` | Offline number series generation (prefix + last sequence per table) |
| `_off_auth_cache` | Cached offline credentials with expiry (72h window), user groups, device binding |
| `_off_model_registry` | Schema cache: maps model names to table names and field definitions received from server |

## Encryption

SQLite encryption uses SQLCipher. Enable at build time:

```bash
# Install SQLCipher first
# macOS: brew install sqlcipher
# Ubuntu: apt install libsqlcipher-dev
# Windows: vcpkg install sqlcipher

BITCODE_DB_KEY=your-secret-key cargo tauri build --features encryption
```

The `BITCODE_DB_KEY` environment variable sets the encryption key. Without it, encryption builds fall back to plain SQLite. The key is passed directly in the connection string (`sqlite:bitcode.db?key=...`).

## Architecture

```
Stencil Components (@bitcode/components)
        |
  offline-store.ts          <-- intercept layer
        |
  bc-native.ts bridge       <-- 13 native methods
        |
  Tauri IPC (window.__TAURI__)
        |
  Rust Plugins
  ├── tauri-plugin-sql (SQLite + migrations)
  ├── tauri-plugin-fs (file system)
  ├── tauri-plugin-notification (push notifications)
  ├── tauri-plugin-barcode-scanner (mobile, optional)
  └── tauri-plugin-biometric (mobile, optional)
```

The `bc-native.ts` bridge lives in the components package, not here. It checks for `window.__TAURI__` (enabled by `withGlobalTauri: true` in tauri.conf.json) and routes native capability calls to Tauri IPC. When running in a browser without Tauri, it falls back to Web API equivalents.

The `offline-store.ts` intercept layer sits between Stencil components and the data-fetcher. For models marked `mode: "offline"`, it routes CRUD to `BcNative.dbSelect()` / `BcNative.dbExecute()` (local SQLite). For online models, it passes through to normal `fetch()`. Existing components are unaware of which path is used.

### Cargo Dependencies

| Crate | Version | Purpose |
|-------|---------|---------|
| `tauri` | 2 | App framework, window management, IPC |
| `tauri-plugin-sql` | 2 (sqlite feature) | SQLite database with migration support |
| `tauri-plugin-fs` | 2 | File system read/write |
| `tauri-plugin-notification` | 2 | Push notifications |
| `tauri-plugin-barcode-scanner` | 2 (optional) | Barcode/QR scanning (mobile) |
| `tauri-plugin-biometric` | 2 (optional) | Fingerprint/Face ID auth (mobile) |
| `serde` / `serde_json` | 1 | JSON serialization |

### Feature Flags

| Flag | What it enables |
|------|-----------------|
| `mobile-plugins` | `tauri-plugin-barcode-scanner` + `tauri-plugin-biometric` |
| `encryption` | SQLCipher connection string handling via `BITCODE_DB_KEY` |

Neither flag is enabled by default. Desktop builds need neither. Mobile builds should enable `mobile-plugins`. Encryption is opt-in everywhere.

## Related Repositories

| Repo | Description |
|------|-------------|
| [go-json](https://github.com/bitcode-framework/go-json) | JSON/JSONC programming language engine |
| [go-json-runtimes](https://github.com/bitcode-framework/go-json-runtimes) | Script runtime engines for go-json (Goja, QuickJS, Yaegi, Node.js, Python) |
| [ui-stencil-web-components](https://github.com/bitcode-framework/ui-stencil-web-components) | 119 Stencil Web Components for enterprise UIs |
| [ui-tauri](https://github.com/bitcode-framework/ui-tauri) | This repo. Tauri 2.0 native shell wrapping the component library |

## License

MIT
