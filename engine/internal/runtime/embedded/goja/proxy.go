package goja_runtime

import (
	"fmt"
	"time"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
	"github.com/bitcode-framework/bitcode/internal/runtime/embedded"
)

func (v *GojaVM) buildBitcodeObject(bc *bridge.Context) map[string]any {
	return map[string]any{
		"model":   func(name string) any { return v.createModelProxy(bc, name) },
		"session": bc.Session(),
		"db": map[string]any{
			"query":   func(sql string, args ...any) (any, error) { return bc.DB().Query(sql, args...) },
			"execute": func(sql string, args ...any) (any, error) { return bc.DB().Execute(sql, args...) },
		},
		"http": map[string]any{
			"get":    func(url string, opts ...map[string]any) (any, error) { return bc.HTTP().Get(url, embedded.ParseHTTPOpts(firstMap(opts))) },
			"post":   func(url string, opts ...map[string]any) (any, error) { return bc.HTTP().Post(url, embedded.ParseHTTPOpts(firstMap(opts))) },
			"put":    func(url string, opts ...map[string]any) (any, error) { return bc.HTTP().Put(url, embedded.ParseHTTPOpts(firstMap(opts))) },
			"patch":  func(url string, opts ...map[string]any) (any, error) { return bc.HTTP().Patch(url, embedded.ParseHTTPOpts(firstMap(opts))) },
			"delete": func(url string, opts ...map[string]any) (any, error) { return bc.HTTP().Delete(url, embedded.ParseHTTPOpts(firstMap(opts))) },
		},
		"cache": map[string]any{
			"get": func(key string) (any, error) { return bc.Cache().Get(key) },
			"set": func(key string, val any, opts ...map[string]any) error {
				return bc.Cache().Set(key, val, embedded.ParseCacheOpts(firstMap(opts)))
			},
			"del": func(key string) error { return bc.Cache().Del(key) },
		},
		"fs": map[string]any{
			"read":   func(p string) (string, error) { return bc.FS().Read(p) },
			"write":  func(p, content string) error { return bc.FS().Write(p, content) },
			"exists": func(p string) (bool, error) { return bc.FS().Exists(p) },
			"list":   func(p string) ([]string, error) { return bc.FS().List(p) },
			"mkdir":  func(p string) error { return bc.FS().Mkdir(p) },
			"remove": func(p string) error { return bc.FS().Remove(p) },
		},
		"env":    func(key string) (string, error) { return bc.Env(key) },
		"config": func(key string) any { return bc.Config(key) },
		"log":    func(level, msg string, data ...map[string]any) { bc.Log(level, msg, data...) },
		"emit":   func(event string, data map[string]any) error { return bc.Emit(event, data) },
		"call":   func(process string, input map[string]any) (any, error) { return bc.Call(process, input) },
		"t":      func(key string) string { return bc.T(key) },
		"exec": func(cmd string, args []string, opts ...map[string]any) (any, error) {
			return bc.Exec(cmd, args, embedded.ParseExecOpts(firstMap(opts)))
		},
		"email": map[string]any{
			"send": func(opts map[string]any) error { return bc.Email().Send(embedded.ParseEmailOpts(opts)) },
		},
		"notify": map[string]any{
			"send":      func(opts map[string]any) error { return bc.Notify().Send(embedded.ParseNotifyOpts(opts)) },
			"broadcast": func(channel string, data map[string]any) error { return bc.Notify().Broadcast(channel, data) },
		},
		"storage": map[string]any{
			"upload":   func(opts map[string]any) (any, error) { return bc.Storage().Upload(embedded.ParseUploadOpts(opts)) },
			"url":      func(id string) (string, error) { return bc.Storage().URL(id) },
			"download": func(id string) ([]byte, error) { return bc.Storage().Download(id) },
			"delete":   func(id string) error { return bc.Storage().Delete(id) },
		},
		"security": map[string]any{
			"permissions": func(model string) (any, error) { return bc.Security().Permissions(model) },
			"hasGroup":    func(group string) (bool, error) { return bc.Security().HasGroup(group) },
			"groups":      func() ([]string, error) { return bc.Security().Groups() },
		},
		"audit": map[string]any{
			"log": func(opts map[string]any) error { return bc.Audit().Log(embedded.ParseAuditOpts(opts)) },
		},
		"crypto": map[string]any{
			"encrypt": func(plaintext string) (string, error) { return bc.Crypto().Encrypt(plaintext) },
			"decrypt": func(ciphertext string) (string, error) { return bc.Crypto().Decrypt(ciphertext) },
			"hash":    func(value string) (string, error) { return bc.Crypto().Hash(value) },
			"verify":  func(value, hash string) (bool, error) { return bc.Crypto().Verify(value, hash) },
		},
		"execution": map[string]any{
			"search":  func(opts map[string]any) (any, error) { return bc.Execution().Search(parseExecSearchOpts(opts)) },
			"get":     func(id string) (any, error) { return bc.Execution().Get(id) },
			"current": func() any { return bc.Execution().Current() },
			"cancel":  func(id string) error { return bc.Execution().Cancel(id) },
		},
		"tx": map[string]any{
			"begin": func(opts ...map[string]any) error {
				db := bc.GormDB()
				if db == nil {
					return fmt.Errorf("database not available for transactions")
				}
				v.txMu.Lock()
				defer v.txMu.Unlock()
				if v.txGormTx != nil {
					return fmt.Errorf("transaction already active — commit or rollback first")
				}
				gormTx := db.Begin()
				if gormTx.Error != nil {
					return gormTx.Error
				}
				txCtx := bc.CloneWithGormTx(gormTx)
				v.txOriginalBC = bc
				v.txGormTx = gormTx
				timeout := parseTxBeginTimeout(firstMap(opts))
				if timeout > 0 {
					v.txTimeout = time.AfterFunc(timeout, func() {
						v.txMu.Lock()
						defer v.txMu.Unlock()
						if v.txGormTx != nil {
							v.txGormTx.Rollback()
							v.txGormTx = nil
							v.txOriginalBC = nil
							v.txTimeout = nil
						}
					})
				}
				v.rt.Set("bitcode", v.buildBitcodeObject(txCtx))
				return nil
			},
			"commit": func() error {
				v.txMu.Lock()
				defer v.txMu.Unlock()
				if v.txGormTx == nil {
					return fmt.Errorf("no active transaction to commit")
				}
				if v.txTimeout != nil {
					v.txTimeout.Stop()
					v.txTimeout = nil
				}
				err := v.txGormTx.Commit().Error
				v.rt.Set("bitcode", v.buildBitcodeObject(v.txOriginalBC))
				v.txGormTx = nil
				v.txOriginalBC = nil
				return err
			},
			"rollback": func() error {
				v.txMu.Lock()
				defer v.txMu.Unlock()
				if v.txGormTx == nil {
					return fmt.Errorf("no active transaction to rollback")
				}
				if v.txTimeout != nil {
					v.txTimeout.Stop()
					v.txTimeout = nil
				}
				err := v.txGormTx.Rollback().Error
				v.rt.Set("bitcode", v.buildBitcodeObject(v.txOriginalBC))
				v.txGormTx = nil
				v.txOriginalBC = nil
				return err
			},
		},
	}
}

func (v *GojaVM) createModelProxy(bc *bridge.Context, name string) map[string]any {
	model := bc.Model(name)
	return map[string]any{
		"search": func(opts ...map[string]any) (any, error) {
			return model.Search(embedded.ParseSearchOpts(firstMap(opts)))
		},
		"get": func(id string) (any, error) { return model.Get(id) },
		"create": func(data map[string]any) (any, error) { return model.Create(data) },
		"write":  func(id string, data map[string]any) error { return model.Write(id, data) },
		"delete": func(id string) error { return model.Delete(id) },
		"count": func(opts ...map[string]any) (int64, error) {
			return model.Count(embedded.ParseSearchOpts(firstMap(opts)))
		},
		"sum": func(field string, opts ...map[string]any) (float64, error) {
			return model.Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
		},
		"upsert": func(data map[string]any, uniqueFields []string) (any, error) {
			return model.Upsert(data, uniqueFields)
		},
		"createMany": func(records []map[string]any) (any, error) { return model.CreateMany(records) },
		"writeMany": func(ids []string, data map[string]any) (any, error) {
			return model.WriteMany(ids, data)
		},
		"deleteMany": func(ids []string) (any, error) { return model.DeleteMany(ids) },
		"upsertMany": func(records []map[string]any, uniqueFields []string) (any, error) {
			return model.UpsertMany(records, uniqueFields)
		},
		"addRelation":    func(id, field string, relatedIDs []string) error { return model.AddRelation(id, field, relatedIDs) },
		"removeRelation": func(id, field string, relatedIDs []string) error { return model.RemoveRelation(id, field, relatedIDs) },
		"setRelation":    func(id, field string, relatedIDs []string) error { return model.SetRelation(id, field, relatedIDs) },
		"loadRelation":   func(id, field string) (any, error) { return model.LoadRelation(id, field) },
		"sudo": func() any {
			sudo := model.Sudo()
			return v.createSudoModelProxy(sudo)
		},
	}
}

func (v *GojaVM) createSudoModelProxy(sudo bridge.SudoModelHandle) map[string]any {
	return map[string]any{
		"search": func(opts ...map[string]any) (any, error) {
			return sudo.Search(embedded.ParseSearchOpts(firstMap(opts)))
		},
		"get":    func(id string) (any, error) { return sudo.Get(id) },
		"create": func(data map[string]any) (any, error) { return sudo.Create(data) },
		"write":  func(id string, data map[string]any) error { return sudo.Write(id, data) },
		"delete": func(id string) error { return sudo.Delete(id) },
		"count": func(opts ...map[string]any) (int64, error) {
			return sudo.Count(embedded.ParseSearchOpts(firstMap(opts)))
		},
		"sum": func(field string, opts ...map[string]any) (float64, error) {
			return sudo.Sum(field, embedded.ParseSearchOpts(firstMap(opts)))
		},
		"upsert": func(data map[string]any, uniqueFields []string) (any, error) {
			return sudo.Upsert(data, uniqueFields)
		},
		"createMany":     func(records []map[string]any) (any, error) { return sudo.CreateMany(records) },
		"writeMany":      func(ids []string, data map[string]any) (any, error) { return sudo.WriteMany(ids, data) },
		"deleteMany":     func(ids []string) (any, error) { return sudo.DeleteMany(ids) },
		"upsertMany":     func(records []map[string]any, uniqueFields []string) (any, error) { return sudo.UpsertMany(records, uniqueFields) },
		"addRelation":    func(id, field string, relatedIDs []string) error { return sudo.AddRelation(id, field, relatedIDs) },
		"removeRelation": func(id, field string, relatedIDs []string) error { return sudo.RemoveRelation(id, field, relatedIDs) },
		"setRelation":    func(id, field string, relatedIDs []string) error { return sudo.SetRelation(id, field, relatedIDs) },
		"loadRelation":   func(id, field string) (any, error) { return sudo.LoadRelation(id, field) },
		"hardDelete":     func(id string) error { return sudo.HardDelete(id) },
		"hardDeleteMany": func(ids []string) (any, error) { return sudo.HardDeleteMany(ids) },
		"withTenant": func(tenantID string) any {
			return v.createSudoModelProxy(sudo.WithTenant(tenantID))
		},
		"skipValidation": func() any {
			return v.createSudoModelProxy(sudo.SkipValidation())
		},
	}
}

func parseExecSearchOpts(raw map[string]any) bridge.ExecutionSearchOptions {
	opts := bridge.ExecutionSearchOptions{}
	if raw == nil {
		return opts
	}
	if process, ok := raw["process"].(string); ok {
		opts.Process = process
	}
	if status, ok := raw["status"].(string); ok {
		opts.Status = status
	}
	if userID, ok := raw["userId"].(string); ok {
		opts.UserID = userID
	}
	if limit, ok := raw["limit"]; ok {
		opts.Limit = embedded.ToInt(limit)
	}
	if offset, ok := raw["offset"]; ok {
		opts.Offset = embedded.ToInt(offset)
	}
	if order, ok := raw["order"].(string); ok {
		opts.Order = order
	}
	return opts
}

func firstMap(opts []map[string]any) map[string]any {
	if len(opts) > 0 {
		return opts[0]
	}
	return nil
}

const defaultGojaTxTimeout = 30 * time.Second

func parseTxBeginTimeout(opts map[string]any) time.Duration {
	if opts == nil {
		return defaultGojaTxTimeout
	}
	raw, ok := opts["timeout"]
	if !ok {
		return defaultGojaTxTimeout
	}
	switch v := raw.(type) {
	case float64:
		if v <= 0 {
			return 0
		}
		return time.Duration(v) * time.Second
	case int:
		if v <= 0 {
			return 0
		}
		return time.Duration(v) * time.Second
	case int64:
		if v <= 0 {
			return 0
		}
		return time.Duration(v) * time.Second
	case string:
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 {
			return defaultGojaTxTimeout
		}
		return d
	default:
		return defaultGojaTxTimeout
	}
}

