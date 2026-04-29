package plugin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bitcode-framework/bitcode/internal/runtime/bridge"
)

type BridgeHandler struct{}

func (h *BridgeHandler) Handle(ctx context.Context, bc *bridge.Context, method string, rawParams map[string]any) (any, *bridge.BridgeError) {
	parts := strings.SplitN(method, ".", 2)
	namespace := parts[0]

	decodeBinaryFields(rawParams)

	switch namespace {
	case "model":
		return h.handleModel(ctx, bc, method, rawParams)
	case "db":
		return h.handleDB(ctx, bc, method, rawParams)
	case "http":
		return h.handleHTTP(ctx, bc, rawParams)
	case "cache":
		return h.handleCache(ctx, bc, method, rawParams)
	case "fs":
		return h.handleFS(ctx, bc, method, rawParams)
	case "email":
		return h.handleEmail(ctx, bc, method, rawParams)
	case "notify":
		return h.handleNotify(ctx, bc, method, rawParams)
	case "storage":
		return h.handleStorage(ctx, bc, method, rawParams)
	case "security":
		return h.handleSecurity(ctx, bc, method, rawParams)
	case "audit":
		return h.handleAudit(ctx, bc, method, rawParams)
	case "crypto":
		return h.handleCrypto(ctx, bc, method, rawParams)
	case "execution":
		return h.handleExecution(ctx, bc, method, rawParams)
	case "env":
		return h.handleEnv(ctx, bc, rawParams)
	case "config":
		return h.handleConfig(ctx, bc, rawParams)
	case "tx":
		return nil, bridge.NewError("TX_HANDLED_BY_MANAGER", "tx.* methods are handled by the manager")
	case "log":
		return h.handleLog(ctx, bc, rawParams)
	case "emit":
		return h.handleEmit(ctx, bc, rawParams)
	case "call":
		return h.handleCall(ctx, bc, rawParams)
	case "exec":
		return h.handleExec(ctx, bc, rawParams)
	case "t":
		return h.handleTranslate(ctx, bc, rawParams)
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown bridge method: %s", method)
	}
}

func (h *BridgeHandler) handleModel(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	modelName := getString(params, "model")
	if modelName == "" {
		return nil, bridge.NewError(bridge.ErrValidation, "model name is required")
	}

	sudo := getBool(params, "sudo")
	var handle bridge.ModelHandle
	if sudo {
		handle = bc.Model(modelName).Sudo()
	} else {
		handle = bc.Model(modelName)
	}

	action := strings.TrimPrefix(method, "model.")

	switch action {
	case "search":
		opts := parseSearchOptions(getMap(params, "opts"))
		return wrapResult(handle.Search(opts))
	case "get":
		id := getString(params, "id")
		return wrapResult(handle.Get(id))
	case "create":
		data := getMap(params, "data")
		return wrapResult(handle.Create(data))
	case "write":
		id := getString(params, "id")
		data := getMap(params, "data")
		return nil, toBridgeError(handle.Write(id, data))
	case "delete":
		id := getString(params, "id")
		return nil, toBridgeError(handle.Delete(id))
	case "count":
		opts := parseSearchOptions(getMap(params, "opts"))
		return wrapResult(handle.Count(opts))
	case "sum":
		field := getString(params, "field")
		opts := parseSearchOptions(getMap(params, "opts"))
		return wrapResult(handle.Sum(field, opts))
	case "upsert":
		data := getMap(params, "data")
		unique := getStringSlice(params, "unique")
		return wrapResult(handle.Upsert(data, unique))
	case "createMany":
		records := getMapSlice(params, "records")
		return wrapResult(handle.CreateMany(records))
	case "writeMany":
		ids := getStringSlice(params, "ids")
		data := getMap(params, "data")
		return wrapResult(handle.WriteMany(ids, data))
	case "deleteMany":
		ids := getStringSlice(params, "ids")
		return wrapResult(handle.DeleteMany(ids))
	case "upsertMany":
		records := getMapSlice(params, "records")
		unique := getStringSlice(params, "unique")
		return wrapResult(handle.UpsertMany(records, unique))
	case "addRelation":
		id := getString(params, "id")
		field := getString(params, "field")
		relatedIDs := getStringSlice(params, "relatedIds")
		return nil, toBridgeError(handle.AddRelation(id, field, relatedIDs))
	case "removeRelation":
		id := getString(params, "id")
		field := getString(params, "field")
		relatedIDs := getStringSlice(params, "relatedIds")
		return nil, toBridgeError(handle.RemoveRelation(id, field, relatedIDs))
	case "setRelation":
		id := getString(params, "id")
		field := getString(params, "field")
		relatedIDs := getStringSlice(params, "relatedIds")
		return nil, toBridgeError(handle.SetRelation(id, field, relatedIDs))
	case "loadRelation":
		id := getString(params, "id")
		field := getString(params, "field")
		return wrapResult(handle.LoadRelation(id, field))
	case "hardDelete":
		if !sudo {
			return nil, bridge.NewError(bridge.ErrSudoNotAllowed, "hardDelete requires sudo mode")
		}
		id := getString(params, "id")
		sudoHandle, ok := handle.(bridge.SudoModelHandle)
		if !ok {
			return nil, bridge.NewError(bridge.ErrInternalError, "sudo handle not available")
		}
		return nil, toBridgeError(sudoHandle.HardDelete(id))
	case "hardDeleteMany":
		if !sudo {
			return nil, bridge.NewError(bridge.ErrSudoNotAllowed, "hardDeleteMany requires sudo mode")
		}
		ids := getStringSlice(params, "ids")
		sudoHandle, ok := handle.(bridge.SudoModelHandle)
		if !ok {
			return nil, bridge.NewError(bridge.ErrInternalError, "sudo handle not available")
		}
		return wrapResult(sudoHandle.HardDeleteMany(ids))
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown model method: %s", action)
	}
}

func (h *BridgeHandler) handleDB(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	sql := getString(params, "sql")
	args := getAnySlice(params, "args")

	switch method {
	case "db.query":
		result, err := bc.DB().Query(sql, args...)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return result, nil
	case "db.execute":
		result, err := bc.DB().Execute(sql, args...)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return result, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown db method: %s", method)
	}
}

func (h *BridgeHandler) handleHTTP(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	httpMethod := getString(params, "method")
	url := getString(params, "url")

	opts := &bridge.HTTPOptions{
		Headers: getStringMap(params, "headers"),
		Body:    params["body"],
		Timeout: getInt(params, "timeout"),
	}

	var result *bridge.HTTPResponse
	var err error

	switch httpMethod {
	case "GET":
		result, err = bc.HTTP().Get(url, opts)
	case "POST":
		result, err = bc.HTTP().Post(url, opts)
	case "PUT":
		result, err = bc.HTTP().Put(url, opts)
	case "PATCH":
		result, err = bc.HTTP().Patch(url, opts)
	case "DELETE":
		result, err = bc.HTTP().Delete(url, opts)
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown HTTP method: %s", httpMethod)
	}

	if err != nil {
		return nil, bridge.NewError(bridge.ErrHTTPError, err.Error())
	}
	return result, nil
}

func (h *BridgeHandler) handleCache(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "cache.get":
		key := getString(params, "key")
		val, err := bc.Cache().Get(key)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return val, nil
	case "cache.set":
		key := getString(params, "key")
		value := params["value"]
		var opts *bridge.CacheSetOptions
		if ttl := getInt(params, "ttl"); ttl > 0 {
			opts = &bridge.CacheSetOptions{TTL: ttl}
		}
		if err := bc.Cache().Set(key, value, opts); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return nil, nil
	case "cache.del":
		key := getString(params, "key")
		if err := bc.Cache().Del(key); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return nil, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown cache method: %s", method)
	}
}

func (h *BridgeHandler) handleFS(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	p := getString(params, "path")
	switch method {
	case "fs.read":
		content, err := bc.FS().Read(p)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrFSNotFound, err.Error())
		}
		return content, nil
	case "fs.write":
		content := getString(params, "content")
		if err := bc.FS().Write(p, content); err != nil {
			return nil, bridge.NewError(bridge.ErrFSAccessDenied, err.Error())
		}
		return nil, nil
	case "fs.exists":
		exists, err := bc.FS().Exists(p)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return exists, nil
	case "fs.list":
		entries, err := bc.FS().List(p)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrFSNotFound, err.Error())
		}
		return entries, nil
	case "fs.mkdir":
		if err := bc.FS().Mkdir(p); err != nil {
			return nil, bridge.NewError(bridge.ErrFSAccessDenied, err.Error())
		}
		return nil, nil
	case "fs.remove":
		if err := bc.FS().Remove(p); err != nil {
			return nil, bridge.NewError(bridge.ErrFSAccessDenied, err.Error())
		}
		return nil, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown fs method: %s", method)
	}
}

func (h *BridgeHandler) handleEmail(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	if method != "email.send" {
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown email method: %s", method)
	}
	opts := bridge.EmailOptions{
		To:       getString(params, "to"),
		Subject:  getString(params, "subject"),
		Body:     getString(params, "body"),
		Template: getString(params, "template"),
		Data:     getMap(params, "data"),
	}
	if err := bc.Email().Send(opts); err != nil {
		return nil, bridge.NewError(bridge.ErrEmailNotConfigured, err.Error())
	}
	return nil, nil
}

func (h *BridgeHandler) handleNotify(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "notify.send":
		opts := bridge.NotifyOptions{
			To:      getString(params, "to"),
			Title:   getString(params, "title"),
			Message: getString(params, "message"),
			Type:    getString(params, "type"),
		}
		if err := bc.Notify().Send(opts); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return nil, nil
	case "notify.broadcast":
		channel := getString(params, "channel")
		data := getMap(params, "data")
		if err := bc.Notify().Broadcast(channel, data); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return nil, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown notify method: %s", method)
	}
}

func (h *BridgeHandler) handleStorage(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "storage.upload":
		opts := bridge.UploadOptions{
			Filename: getString(params, "filename"),
			Model:    getString(params, "model"),
			RecordID: getString(params, "recordId"),
		}
		if content, ok := params["content"]; ok {
			if b, ok := content.([]byte); ok {
				opts.Content = b
			}
		}
		att, err := bc.Storage().Upload(opts)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrStorageError, err.Error())
		}
		return att, nil
	case "storage.url":
		id := getString(params, "id")
		url, err := bc.Storage().URL(id)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrStorageError, err.Error())
		}
		return url, nil
	case "storage.download":
		id := getString(params, "id")
		data, err := bc.Storage().Download(id)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrStorageError, err.Error())
		}
		return map[string]any{
			"_type":    "binary",
			"encoding": "base64",
			"data":     base64.StdEncoding.EncodeToString(data),
		}, nil
	case "storage.delete":
		id := getString(params, "id")
		if err := bc.Storage().Delete(id); err != nil {
			return nil, bridge.NewError(bridge.ErrStorageError, err.Error())
		}
		return nil, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown storage method: %s", method)
	}
}

func (h *BridgeHandler) handleSecurity(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "security.permissions":
		model := getString(params, "model")
		perms, err := bc.Security().Permissions(model)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return perms, nil
	case "security.hasGroup":
		group := getString(params, "group")
		has, err := bc.Security().HasGroup(group)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return has, nil
	case "security.groups":
		groups, err := bc.Security().Groups()
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return groups, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown security method: %s", method)
	}
}

func (h *BridgeHandler) handleAudit(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	if method != "audit.log" {
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown audit method: %s", method)
	}
	opts := bridge.AuditOptions{
		Action:   getString(params, "action"),
		Model:    getString(params, "model"),
		RecordID: getString(params, "recordId"),
		Detail:   getString(params, "detail"),
	}
	if err := bc.Audit().Log(opts); err != nil {
		return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
	}
	return nil, nil
}

func (h *BridgeHandler) handleCrypto(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "crypto.encrypt":
		text := getString(params, "text")
		result, err := bc.Crypto().Encrypt(text)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCryptoError, err.Error())
		}
		return result, nil
	case "crypto.decrypt":
		text := getString(params, "text")
		result, err := bc.Crypto().Decrypt(text)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCryptoError, err.Error())
		}
		return result, nil
	case "crypto.hash":
		value := getString(params, "value")
		result, err := bc.Crypto().Hash(value)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCryptoError, err.Error())
		}
		return result, nil
	case "crypto.verify":
		value := getString(params, "value")
		hash := getString(params, "hash")
		ok, err := bc.Crypto().Verify(value, hash)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrCryptoError, err.Error())
		}
		return ok, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown crypto method: %s", method)
	}
}

func (h *BridgeHandler) handleExecution(ctx context.Context, bc *bridge.Context, method string, params map[string]any) (any, *bridge.BridgeError) {
	switch method {
	case "execution.search":
		opts := bridge.ExecutionSearchOptions{
			Process: getString(params, "process"),
			Status:  getString(params, "status"),
			UserID:  getString(params, "userId"),
			Limit:   getInt(params, "limit"),
			Offset:  getInt(params, "offset"),
			Order:   getString(params, "order"),
		}
		result, err := bc.Execution().Search(opts)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return result, nil
	case "execution.get":
		id := getString(params, "id")
		result, err := bc.Execution().Get(id)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return result, nil
	case "execution.current":
		info := bc.Execution().Current()
		return info, nil
	case "execution.retry":
		id := getString(params, "id")
		result, err := bc.Execution().Retry(id)
		if err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return result, nil
	case "execution.cancel":
		id := getString(params, "id")
		if err := bc.Execution().Cancel(id); err != nil {
			return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
		}
		return nil, nil
	default:
		return nil, bridge.NewErrorf("UNKNOWN_METHOD", "unknown execution method: %s", method)
	}
}

func (h *BridgeHandler) handleEnv(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	key := getString(params, "key")
	val, err := bc.Env(key)
	if err != nil {
		if be, ok := err.(*bridge.BridgeError); ok {
			return nil, be
		}
		return nil, bridge.NewError(bridge.ErrEnvAccessDenied, err.Error())
	}
	return val, nil
}

func (h *BridgeHandler) handleConfig(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	key := getString(params, "key")
	return bc.Config(key), nil
}

func (h *BridgeHandler) handleLog(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	level := getString(params, "level")
	msg := getString(params, "msg")
	if level == "" {
		level = "info"
	}
	data := getMap(params, "data")
	if data != nil {
		bc.Log(level, msg, data)
	} else {
		bc.Log(level, msg)
	}
	return nil, nil
}

func (h *BridgeHandler) handleEmit(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	event := getString(params, "event")
	data := getMap(params, "data")
	if err := bc.Emit(event, data); err != nil {
		return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
	}
	return nil, nil
}

func (h *BridgeHandler) handleCall(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	process := getString(params, "process")
	input := getMap(params, "input")
	result, err := bc.Call(process, input)
	if err != nil {
		if be, ok := err.(*bridge.BridgeError); ok {
			return nil, be
		}
		return nil, bridge.NewError(bridge.ErrInternalError, err.Error())
	}
	return result, nil
}

func (h *BridgeHandler) handleExec(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	cmd := getString(params, "cmd")
	args := getStringSlice(params, "args")
	var opts *bridge.ExecOptions
	if cwd := getString(params, "cwd"); cwd != "" {
		opts = &bridge.ExecOptions{Cwd: cwd}
	}
	if timeout := getInt(params, "timeout"); timeout > 0 {
		if opts == nil {
			opts = &bridge.ExecOptions{}
		}
		opts.Timeout = timeout
	}
	result, err := bc.Exec(cmd, args, opts)
	if err != nil {
		if be, ok := err.(*bridge.BridgeError); ok {
			return nil, be
		}
		return nil, bridge.NewError(bridge.ErrExecDenied, err.Error())
	}
	return result, nil
}

func (h *BridgeHandler) handleTranslate(ctx context.Context, bc *bridge.Context, params map[string]any) (any, *bridge.BridgeError) {
	key := getString(params, "key")
	return bc.T(key), nil
}

func decodeBinaryFields(params map[string]any) {
	for key, val := range params {
		if m, ok := val.(map[string]any); ok {
			if m["_type"] == "binary" {
				if dataStr, ok := m["data"].(string); ok {
					decoded, err := base64.StdEncoding.DecodeString(dataStr)
					if err == nil {
						params[key] = decoded
					}
				}
			} else {
				decodeBinaryFields(m)
			}
		}
	}
}

func wrapResult[T any](result T, err error) (any, *bridge.BridgeError) {
	if err != nil {
		return nil, toBridgeError(err)
	}
	return result, nil
}

func toBridgeError(err error) *bridge.BridgeError {
	if err == nil {
		return nil
	}
	if be, ok := err.(*bridge.BridgeError); ok {
		return be
	}
	return bridge.NewError(bridge.ErrInternalError, err.Error())
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		}
	}
	return 0
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if mm, ok := v.(map[string]any); ok {
			return mm
		}
	}
	return nil
}

func getStringMap(m map[string]any, key string) map[string]string {
	raw := getMap(m, key)
	if raw == nil {
		return nil
	}
	result := make(map[string]string, len(raw))
	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func getStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				result = append(result, fmt.Sprintf("%v", item))
			}
			return result
		}
	}
	return nil
}

func getAnySlice(m map[string]any, key string) []any {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			return arr
		}
	}
	return nil
}

func getMapSlice(m map[string]any, key string) []map[string]any {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]map[string]any, 0, len(arr))
			for _, item := range arr {
				if mm, ok := item.(map[string]any); ok {
					result = append(result, mm)
				}
			}
			return result
		}
	}
	return nil
}

func parseSearchOptions(m map[string]any) bridge.SearchOptions {
	if m == nil {
		return bridge.SearchOptions{}
	}
	opts := bridge.SearchOptions{
		Order:  getString(m, "order"),
		Limit:  getInt(m, "limit"),
		Offset: getInt(m, "offset"),
	}
	opts.Fields = getStringSlice(m, "fields")
	opts.Include = getStringSlice(m, "include")
	if domain, ok := m["domain"]; ok {
		if arr, ok := domain.([]any); ok {
			for _, item := range arr {
				if cond, ok := item.([]any); ok {
					opts.Domain = append(opts.Domain, cond)
				}
			}
		}
	}
	return opts
}
