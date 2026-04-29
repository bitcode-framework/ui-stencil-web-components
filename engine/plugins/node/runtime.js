'use strict';

const readline = require('readline');
const path = require('path');
const vm = require('vm');
const fs = require('fs');
const Module = require('module');

const pendingBridgeRequests = new Map();
let nextBridgeId = 1;

function sendToGo(message) {
  process.stdout.write(JSON.stringify(message) + '\n');
}

function serializeParams(params) {
  return JSON.parse(JSON.stringify(params, function(key, value) {
    if (Buffer.isBuffer(this[key])) {
      return { _type: 'binary', encoding: 'base64', data: this[key].toString('base64') };
    }
    return value;
  }));
}

function bridgeCall(method, params, txId) {
  return new Promise((resolve, reject) => {
    const id = nextBridgeId++;
    pendingBridgeRequests.set(id, { resolve, reject });
    const msg = { type: 'bridge_request', id, method, params: serializeParams(params || {}) };
    if (txId) msg.txId = txId;
    sendToGo(msg);
  });
}

function createBitcodeProxy(session, securityRules, moduleDir) {
  function createModelHandle(modelName, sudoMode, tenantOverride, skipVal) {
    const baseParams = { model: modelName };
    if (sudoMode) baseParams.sudo = true;
    if (tenantOverride) baseParams.tenant = tenantOverride;
    if (skipVal) baseParams.skipValidation = true;

    const handle = {
      search: (opts) => bridgeCall('model.search', { ...baseParams, opts: opts || {} }),
      get: (id, opts) => bridgeCall('model.get', { ...baseParams, id, opts }),
      create: (data) => bridgeCall('model.create', { ...baseParams, data }),
      write: (id, data) => bridgeCall('model.write', { ...baseParams, id, data }),
      delete: (id) => bridgeCall('model.delete', { ...baseParams, id }),
      count: (opts) => bridgeCall('model.count', { ...baseParams, opts: opts || {} }),
      sum: (field, opts) => bridgeCall('model.sum', { ...baseParams, field, opts: opts || {} }),
      upsert: (data, unique) => bridgeCall('model.upsert', { ...baseParams, data, unique }),
      createMany: (records) => bridgeCall('model.createMany', { ...baseParams, records }),
      writeMany: (ids, data) => bridgeCall('model.writeMany', { ...baseParams, ids, data }),
      deleteMany: (ids) => bridgeCall('model.deleteMany', { ...baseParams, ids }),
      upsertMany: (records, unique) => bridgeCall('model.upsertMany', { ...baseParams, records, unique }),
      addRelation: (id, field, ids) => bridgeCall('model.addRelation', { ...baseParams, id, field, relatedIds: ids }),
      removeRelation: (id, field, ids) => bridgeCall('model.removeRelation', { ...baseParams, id, field, relatedIds: ids }),
      setRelation: (id, field, ids) => bridgeCall('model.setRelation', { ...baseParams, id, field, relatedIds: ids }),
      loadRelation: (id, field) => bridgeCall('model.loadRelation', { ...baseParams, id, field }),
    };

    if (!sudoMode) {
      handle.sudo = () => createModelHandle(modelName, true, null, false);
    } else {
      handle.hardDelete = (id) => bridgeCall('model.hardDelete', { ...baseParams, id });
      handle.hardDeleteMany = (ids) => bridgeCall('model.hardDeleteMany', { ...baseParams, ids });
      handle.withTenant = (tid) => createModelHandle(modelName, true, tid, skipVal);
      handle.skipValidation = () => createModelHandle(modelName, true, tenantOverride, true);
    }

    return handle;
  }

  const scriptConsole = {
    log: (...args) => bridgeCall('log', { level: 'info', msg: args.map(String).join(' ') }),
    warn: (...args) => bridgeCall('log', { level: 'warn', msg: args.map(String).join(' ') }),
    error: (...args) => bridgeCall('log', { level: 'error', msg: args.map(String).join(' ') }),
    debug: (...args) => bridgeCall('log', { level: 'debug', msg: args.map(String).join(' ') }),
  };

  const bitcode = {
    model: (name) => createModelHandle(name, false, null, false),

    db: {
      query: (sql, ...args) => bridgeCall('db.query', { sql, args }),
      execute: (sql, ...args) => bridgeCall('db.execute', { sql, args }),
    },

    http: {
      get: (url, opts) => bridgeCall('http.request', { method: 'GET', url, ...(opts || {}) }),
      post: (url, opts) => bridgeCall('http.request', { method: 'POST', url, ...(opts || {}) }),
      put: (url, opts) => bridgeCall('http.request', { method: 'PUT', url, ...(opts || {}) }),
      patch: (url, opts) => bridgeCall('http.request', { method: 'PATCH', url, ...(opts || {}) }),
      delete: (url, opts) => bridgeCall('http.request', { method: 'DELETE', url, ...(opts || {}) }),
    },

    cache: {
      get: (key) => bridgeCall('cache.get', { key }),
      set: (key, value, opts) => bridgeCall('cache.set', { key, value, ...(opts || {}) }),
      del: (key) => bridgeCall('cache.del', { key }),
    },

    env: (key) => bridgeCall('env.get', { key }),
    session: session,
    config: (key) => bridgeCall('config.get', { key }),
    log: (level, msg, data) => bridgeCall('log', { level, msg, data }),
    emit: (event, data) => bridgeCall('emit', { event, data }),
    call: (process, input) => bridgeCall('call', { process, input }),

    fs: {
      read: (p) => bridgeCall('fs.read', { path: p }),
      write: (p, content) => bridgeCall('fs.write', { path: p, content }),
      exists: (p) => bridgeCall('fs.exists', { path: p }),
      list: (p) => bridgeCall('fs.list', { path: p }),
      mkdir: (p) => bridgeCall('fs.mkdir', { path: p }),
      remove: (p) => bridgeCall('fs.remove', { path: p }),
    },

    exec: (cmd, args, opts) => bridgeCall('exec', { cmd, args, ...(opts || {}) }),

    email: {
      send: (opts) => bridgeCall('email.send', opts),
    },

    notify: {
      send: (opts) => bridgeCall('notify.send', opts),
      broadcast: (channel, data) => bridgeCall('notify.broadcast', { channel, data }),
    },

    storage: {
      upload: (opts) => bridgeCall('storage.upload', opts),
      url: (id) => bridgeCall('storage.url', { id }),
      download: (id) => bridgeCall('storage.download', { id }),
      delete: (id) => bridgeCall('storage.delete', { id }),
    },

    t: (key) => bridgeCall('t', { key }),

    security: {
      permissions: (model) => bridgeCall('security.permissions', { model }),
      hasGroup: (group) => bridgeCall('security.hasGroup', { group }),
      groups: () => bridgeCall('security.groups', {}),
    },

    audit: {
      log: (opts) => bridgeCall('audit.log', opts),
    },

    crypto: {
      encrypt: (text) => bridgeCall('crypto.encrypt', { text }),
      decrypt: (text) => bridgeCall('crypto.decrypt', { text }),
      hash: (value) => bridgeCall('crypto.hash', { value }),
      verify: (value, hash) => bridgeCall('crypto.verify', { value, hash }),
    },

    execution: {
      search: (opts) => bridgeCall('execution.search', opts),
      get: (id, opts) => bridgeCall('execution.get', { id, ...opts }),
      current: () => bridgeCall('execution.current', {}),
      retry: (id) => bridgeCall('execution.retry', { id }),
      cancel: (id) => bridgeCall('execution.cancel', { id }),
    },

    tx: async (fn) => {
      const { txId } = await bridgeCall('tx.begin', {});
      const txBitcode = Object.create(bitcode);
      const origModel = bitcode.model;
      const txBridge = (method, params) => bridgeCall(method, params, txId);

      txBitcode.model = (name) => {
        function createTxModelHandle(modelName, sudoMode, tenantOverride, skipVal) {
          const baseParams = { model: modelName };
          if (sudoMode) baseParams.sudo = true;
          if (tenantOverride) baseParams.tenant = tenantOverride;
          if (skipVal) baseParams.skipValidation = true;

          const handle = {
            search: (opts) => txBridge('model.search', { ...baseParams, opts: opts || {} }),
            get: (id, opts) => txBridge('model.get', { ...baseParams, id, opts }),
            create: (data) => txBridge('model.create', { ...baseParams, data }),
            write: (id, data) => txBridge('model.write', { ...baseParams, id, data }),
            delete: (id) => txBridge('model.delete', { ...baseParams, id }),
            count: (opts) => txBridge('model.count', { ...baseParams, opts: opts || {} }),
            sum: (field, opts) => txBridge('model.sum', { ...baseParams, field, opts: opts || {} }),
            upsert: (data, unique) => txBridge('model.upsert', { ...baseParams, data, unique }),
            createMany: (records) => txBridge('model.createMany', { ...baseParams, records }),
            writeMany: (ids, data) => txBridge('model.writeMany', { ...baseParams, ids, data }),
            deleteMany: (ids) => txBridge('model.deleteMany', { ...baseParams, ids }),
            upsertMany: (records, unique) => txBridge('model.upsertMany', { ...baseParams, records, unique }),
            addRelation: (id, field, ids) => txBridge('model.addRelation', { ...baseParams, id, field, relatedIds: ids }),
            removeRelation: (id, field, ids) => txBridge('model.removeRelation', { ...baseParams, id, field, relatedIds: ids }),
            setRelation: (id, field, ids) => txBridge('model.setRelation', { ...baseParams, id, field, relatedIds: ids }),
            loadRelation: (id, field) => txBridge('model.loadRelation', { ...baseParams, id, field }),
          };
          if (!sudoMode) {
            handle.sudo = () => createTxModelHandle(modelName, true, null, false);
          } else {
            handle.hardDelete = (id) => txBridge('model.hardDelete', { ...baseParams, id });
            handle.hardDeleteMany = (ids) => txBridge('model.hardDeleteMany', { ...baseParams, ids });
            handle.withTenant = (tid) => createTxModelHandle(modelName, true, tid, skipVal);
            handle.skipValidation = () => createTxModelHandle(modelName, true, tenantOverride, true);
          }
          return handle;
        }
        return createTxModelHandle(name, false, null, false);
      };

      txBitcode.db = {
        query: (sql, ...args) => txBridge('db.query', { sql, args }),
        execute: (sql, ...args) => txBridge('db.execute', { sql, args }),
      };

      try {
        const result = await fn(txBitcode);
        await bridgeCall('tx.commit', { txId });
        return result;
      } catch (e) {
        await bridgeCall('tx.rollback', { txId }).catch(() => {});
        throw e;
      }
    },
  };

  return { bitcode, scriptConsole };
}

function loadScript(scriptPath) {
  const code = fs.readFileSync(scriptPath, 'utf-8');

  const needsTranspile = scriptPath.endsWith('.ts') || scriptPath.endsWith('.tsx')
    || (code.includes('export default') || code.includes('export {') || code.includes('import '));

  if (!needsTranspile) return code;

  if (typeof Bun !== 'undefined' && (scriptPath.endsWith('.ts') || scriptPath.endsWith('.tsx'))) {
    return code;
  }

  try {
    const esbuild = require('esbuild');
    let loader = 'js';
    if (scriptPath.endsWith('.tsx')) loader = 'tsx';
    else if (scriptPath.endsWith('.ts')) loader = 'ts';

    return esbuild.transformSync(code, {
      loader,
      format: 'cjs',
      target: 'node18',
      sourcemap: 'inline',
    }).code;
  } catch (e) {
    if (e.code === 'MODULE_NOT_FOUND') {
      throw new Error(
        'esbuild is required for TypeScript/ESM transpilation. ' +
        'Run: npm install in engine/plugins/node/'
      );
    }
    throw e;
  }
}

const definePlugin = (obj) => obj;

async function executeScript(scriptPath, params, moduleName, session, securityRules) {
  const moduleDir = moduleName ? path.resolve('modules', moduleName) : path.dirname(scriptPath);
  const { bitcode, scriptConsole } = createBitcodeProxy(session, securityRules, moduleDir);

  if (!fs.existsSync(scriptPath)) {
    throw new Error('script not found: ' + scriptPath);
  }

  const code = loadScript(scriptPath);

  const moduleNodeModules = path.resolve(moduleDir, 'node_modules', '_bridge.js');
  const moduleRequire = Module.createRequire(moduleNodeModules);

  const bitcodeSDK = { definePlugin };
  const wrappedRequire = (id) => {
    if (id === '@bitcode/sdk') return bitcodeSDK;
    return moduleRequire(id);
  };

  const sandbox = {
    module: { exports: {} },
    exports: {},
    require: wrappedRequire,
    console: scriptConsole,
    bitcode,
    params,
    ctx: bitcode,
    definePlugin,
    setTimeout, setInterval, clearTimeout, clearInterval,
    Promise, Buffer, URL, URLSearchParams,
    process: { env: {} },
  };

  vm.runInNewContext(code, sandbox, { filename: scriptPath, timeout: 30000 });

  const plugin = sandbox.module.exports.default || sandbox.module.exports;

  if (typeof plugin === 'function') {
    return await plugin(bitcode, params);
  } else if (plugin && typeof plugin.execute === 'function') {
    return await plugin.execute(bitcode, params);
  } else {
    return { executed: true, script: scriptPath };
  }
}

const rl = readline.createInterface({ input: process.stdin, terminal: false });

rl.on('line', async (line) => {
  let message;
  try {
    message = JSON.parse(line);
  } catch (e) {
    sendToGo({ type: 'error', error: 'invalid JSON' });
    return;
  }

  if (message.type === 'execute') {
    const { id, params } = message;
    try {
      const result = await executeScript(
        params.script,
        params.params || {},
        params.module,
        params.session || {},
        params.securityRules || {}
      );
      sendToGo({ type: 'execute_complete', id, result: result || null });
    } catch (e) {
      sendToGo({
        type: 'execute_error', id,
        error: { code: e.code || 'SCRIPT_ERROR', message: e.message, stack: e.stack }
      });
    }
  } else if (message.type === 'bridge_response') {
    const pending = pendingBridgeRequests.get(message.id);
    if (pending) {
      pendingBridgeRequests.delete(message.id);
      if (message.error) {
        const err = new Error(message.error.message);
        err.code = message.error.code;
        err.details = message.error.details;
        err.retryable = message.error.retryable;
        pending.reject(err);
      } else {
        pending.resolve(message.result);
      }
    }
  }
});

process.on('uncaughtException', (err) => {
  process.stderr.write('[plugin:node] uncaught exception: ' + err.message + '\n');
});

process.on('unhandledRejection', (reason) => {
  process.stderr.write('[plugin:node] unhandled rejection: ' + String(reason) + '\n');
});

process.stderr.write('[plugin:node] ready\n');
