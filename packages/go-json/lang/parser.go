package lang

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Parse parses JSONC/JSON input into a Program AST.
func Parse(input []byte) (*Program, error) {
	cleaned := StripComments(input)

	var raw map[string]any
	if err := json.Unmarshal(cleaned, &raw); err != nil {
		return nil, CompileError("JSON_PARSE", "invalid JSON: "+err.Error(), -1)
	}

	return parseProgram(raw, cleaned)
}

// ParseFile reads a file and parses it.
func ParseFile(path string) (*Program, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, CompileError("FILE_READ", "cannot read file: "+err.Error(), -1)
	}
	return Parse(data)
}

func parseProgram(raw map[string]any, cleanedJSON []byte) (*Program, error) {
	prog := &Program{
		Functions: make(map[string]*FuncDef),
	}

	if name, ok := raw["name"].(string); ok {
		prog.Name = name
	}
	if ver, ok := raw["go_json"].(string); ok {
		prog.GoJSON = ver
	}

	parseComment(&prog.NodeMeta, raw)

	// Parse input schema.
	if inputRaw, ok := raw["input"].(map[string]any); ok {
		for name, typ := range inputRaw {
			typStr, _ := typ.(string)
			prog.Input = append(prog.Input, InputField{Name: name, Type: TypeFromJSON(typStr)})
		}
	}

	// Parse imports — design doc uses "import", support "imports" for compat.
	importsRaw, ok := raw["import"].(map[string]any)
	if !ok {
		importsRaw, ok = raw["imports"].(map[string]any)
	}
	if ok {
		for alias, pathRaw := range importsRaw {
			pathStr, ok := pathRaw.(string)
			if !ok {
				return nil, CompileError("INVALID_IMPORT", fmt.Sprintf("import '%s' path must be a string", alias), -1)
			}
			prog.Imports = append(prog.Imports, &ImportDef{
				Alias:    alias,
				Path:     pathStr,
				PathType: detectImportPathType(pathStr),
			})
		}
	}

	// Parse structs.
	if structsRaw, ok := raw["structs"].(map[string]any); ok {
		prog.Structs = make(map[string]*StructDef)
		for name, sRaw := range structsRaw {
			sMap, ok := sRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_STRUCT", fmt.Sprintf("struct '%s' must be an object", name), -1)
			}
			sd, err := parseStructDef(name, sMap, cleanedJSON)
			if err != nil {
				return nil, err
			}
			prog.Structs[name] = sd
		}
	}

	// Parse limits.
	if limitsRaw, ok := raw["limits"].(map[string]any); ok {
		prog.Limits = parseLimits(limitsRaw)
	}

	// Parse functions — need ordered params from raw JSON.
	if funcsRaw, ok := raw["functions"].(map[string]any); ok {
		for name, fRaw := range funcsRaw {
			fMap, ok := fRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_FUNC", fmt.Sprintf("function '%s' must be an object", name), -1)
			}
			fd, err := parseFuncDef(name, fMap, cleanedJSON)
			if err != nil {
				return nil, err
			}
			prog.Functions[name] = fd
		}
	}

	// Parse server config.
	if serverRaw, ok := raw["server"].(map[string]any); ok {
		sc, err := parseServerConfig(serverRaw)
		if err != nil {
			return nil, err
		}
		prog.Server = sc
	}

	// Parse routes.
	if routesRaw, ok := raw["routes"].([]any); ok {
		routes, err := parseRouteConfigs(routesRaw)
		if err != nil {
			return nil, err
		}
		prog.Routes = routes
	}

	// Parse global middleware.
	if mwRaw, ok := raw["middleware"].([]any); ok {
		for _, v := range mwRaw {
			if s, ok := v.(string); ok {
				prog.Middleware = append(prog.Middleware, s)
			}
		}
	}

	// Parse steps.
	if stepsRaw, ok := raw["steps"].([]any); ok {
		steps, err := parseSteps(stepsRaw)
		if err != nil {
			return nil, err
		}
		prog.Steps = steps
	}

	return prog, nil
}

func parseServerConfig(raw map[string]any) (*ServerConfig, error) {
	sc := &ServerConfig{}

	if v, ok := raw["framework"].(string); ok {
		sc.Framework = v
	}
	if v, ok := raw["port"].(float64); ok {
		sc.Port = int(v)
	}
	if v, ok := raw["host"].(string); ok {
		sc.Host = v
	}
	if v, ok := raw["templates"].(string); ok {
		sc.Templates = v
	}
	if v, ok := raw["graceful_shutdown"].(string); ok {
		sc.GracefulShutdown = v
	}
	if v, ok := raw["read_timeout"].(string); ok {
		sc.ReadTimeout = v
	}
	if v, ok := raw["write_timeout"].(string); ok {
		sc.WriteTimeout = v
	}
	if v, ok := raw["max_body_size"].(string); ok {
		sc.MaxBodySize = v
	}

	// Static: string or {dir, prefix}
	switch s := raw["static"].(type) {
	case string:
		sc.Static = s
	case map[string]any:
		cfg := StaticConfig{}
		if d, ok := s["dir"].(string); ok {
			cfg.Dir = d
		}
		if p, ok := s["prefix"].(string); ok {
			cfg.Prefix = p
		}
		sc.Static = cfg
	}

	// CORS
	if corsRaw, ok := raw["cors"].(map[string]any); ok {
		sc.CORS = &CORSConfig{}
		if v, ok := corsRaw["origins"].([]any); ok {
			sc.CORS.Origins = toStringSlice(v)
		}
		if v, ok := corsRaw["methods"].([]any); ok {
			sc.CORS.Methods = toStringSlice(v)
		}
		if v, ok := corsRaw["headers"].([]any); ok {
			sc.CORS.Headers = toStringSlice(v)
		}
		if v, ok := corsRaw["max_age"].(float64); ok {
			sc.CORS.MaxAge = int(v)
		}
	}

	// JWT
	if jwtRaw, ok := raw["jwt"].(map[string]any); ok {
		sc.JWT = parseJWTConfig(jwtRaw)
	}

	// Auth
	if authRaw, ok := raw["auth"].(map[string]any); ok {
		ac, err := parseAuthConfig(authRaw)
		if err != nil {
			return nil, err
		}
		sc.Auth = ac
	}

	// Rate limit
	if rlRaw, ok := raw["rate_limit"].(map[string]any); ok {
		sc.RateLimit = &RateLimitConfig{}
		if v, ok := rlRaw["requests"].(float64); ok {
			sc.RateLimit.Requests = int(v)
		}
		if v, ok := rlRaw["window"].(string); ok {
			sc.RateLimit.Window = v
		}
		if v, ok := rlRaw["by"].(string); ok {
			sc.RateLimit.By = v
		}
	}

	// Error templates
	if etRaw, ok := raw["error_templates"].(map[string]any); ok {
		sc.ErrorTemplates = make(map[string]string)
		for k, v := range etRaw {
			if s, ok := v.(string); ok {
				sc.ErrorTemplates[k] = s
			}
		}
	}

	return sc, nil
}

func parseJWTConfig(raw map[string]any) *JWTConfig {
	jc := &JWTConfig{}
	if v, ok := raw["secret_env"].(string); ok {
		jc.SecretEnv = v
	}
	if v, ok := raw["algorithm"].(string); ok {
		jc.Algorithm = v
	}
	if v, ok := raw["expiry"].(string); ok {
		jc.Expiry = v
	}
	if v, ok := raw["cookie"].(string); ok {
		jc.Cookie = v
	}
	if v, ok := raw["header"].(string); ok {
		jc.Header = v
	}
	if v, ok := raw["prefix"].(string); ok {
		jc.Prefix = v
	}
	if claimsRaw, ok := raw["claims"].(map[string]any); ok {
		jc.Claims = make(map[string]string)
		for k, v := range claimsRaw {
			if s, ok := v.(string); ok {
				jc.Claims[k] = s
			}
		}
	}
	return jc
}

func parseAuthConfig(raw map[string]any) (*AuthConfig, error) {
	ac := &AuthConfig{}
	if v, ok := raw["default"].(string); ok {
		ac.Default = v
	}
	if strategiesRaw, ok := raw["strategies"].(map[string]any); ok {
		ac.Strategies = make(map[string]*StrategyConfig)
		for name, sRaw := range strategiesRaw {
			sMap, ok := sRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_AUTH_STRATEGY",
					fmt.Sprintf("auth strategy '%s' must be an object", name), -1)
			}
			sc := &StrategyConfig{}
			if v, ok := sMap["type"].(string); ok {
				sc.Type = v
			}
			if v, ok := sMap["secret_env"].(string); ok {
				sc.SecretEnv = v
			}
			if v, ok := sMap["algorithm"].(string); ok {
				sc.Algorithm = v
			}
			if v, ok := sMap["expiry"].(string); ok {
				sc.Expiry = v
			}
			if v, ok := sMap["cookie"].(string); ok {
				sc.Cookie = v
			}
			if v, ok := sMap["header"].(string); ok {
				sc.Header = v
			}
			if v, ok := sMap["prefix"].(string); ok {
				sc.Prefix = v
			}
			if v, ok := sMap["query_param"].(string); ok {
				sc.QueryParam = v
			}
			if v, ok := sMap["keys_env"].(string); ok {
				sc.KeysEnv = v
			}
			if v, ok := sMap["users_env"].(string); ok {
				sc.UsersEnv = v
			}
			if v, ok := sMap["realm"].(string); ok {
				sc.Realm = v
			}
			if v, ok := sMap["handler"].(string); ok {
				sc.Handler = v
			}
			if claimsRaw, ok := sMap["claims"].(map[string]any); ok {
				sc.Claims = make(map[string]string)
				for k, v := range claimsRaw {
					if s, ok := v.(string); ok {
						sc.Claims[k] = s
					}
				}
			}
			ac.Strategies[name] = sc
		}
	}
	return ac, nil
}

func parseRouteConfigs(rawRoutes []any) ([]RouteConfig, error) {
	routes := make([]RouteConfig, 0, len(rawRoutes))
	for i, rRaw := range rawRoutes {
		rMap, ok := rRaw.(map[string]any)
		if !ok {
			return nil, CompileError("INVALID_ROUTE",
				fmt.Sprintf("route %d must be an object", i), -1)
		}
		rc, err := parseRouteConfig(rMap)
		if err != nil {
			return nil, err
		}
		routes = append(routes, rc)
	}
	return routes, nil
}

func parseRouteConfig(raw map[string]any) (RouteConfig, error) {
	rc := RouteConfig{}
	if v, ok := raw["method"].(string); ok {
		rc.Method = v
	}
	if v, ok := raw["path"].(string); ok {
		rc.Path = v
	}
	if v, ok := raw["handler"].(string); ok {
		rc.Handler = v
	}
	if v, ok := raw["render"].(string); ok {
		rc.Render = v
	}
	if v, ok := raw["prefix"].(string); ok {
		rc.Prefix = v
	}

	// Middleware: string or []string
	switch mw := raw["middleware"].(type) {
	case string:
		rc.Middleware = []string{mw}
	case []any:
		rc.Middleware = toStringSlice(mw)
	}

	// Nested routes (for groups)
	if nestedRaw, ok := raw["routes"].([]any); ok {
		nested, err := parseRouteConfigs(nestedRaw)
		if err != nil {
			return rc, err
		}
		rc.Routes = nested
	}

	// API annotation
	if apiRaw, ok := raw["api"].(map[string]any); ok {
		api, err := parseAPIAnnotation(apiRaw)
		if err != nil {
			return rc, err
		}
		rc.API = api
	}

	return rc, nil
}

func parseAPIAnnotation(raw map[string]any) (*APIAnnotation, error) {
	api := &APIAnnotation{}
	if v, ok := raw["summary"].(string); ok {
		api.Summary = v
	}
	if v, ok := raw["description"].(string); ok {
		api.Description = v
	}
	if v, ok := raw["tags"].([]any); ok {
		api.Tags = toStringSlice(v)
	}

	// Body
	if bodyRaw, ok := raw["body"].(map[string]any); ok {
		body := &APIBodyAnnotation{}
		if v, ok := bodyRaw["required"].(bool); ok {
			body.Required = v
		}
		if contentRaw, ok := bodyRaw["content"].(map[string]any); ok {
			body.Content = make(map[string]*APIFieldAnnotation)
			for name, fRaw := range contentRaw {
				fMap, ok := fRaw.(map[string]any)
				if !ok {
					continue
				}
				body.Content[name] = parseAPIFieldAnnotation(fMap)
			}
		}
		api.Body = body
	}

	// Query
	if queryRaw, ok := raw["query"].(map[string]any); ok {
		api.Query = make(map[string]*APIParamAnnotation)
		for name, pRaw := range queryRaw {
			pMap, ok := pRaw.(map[string]any)
			if !ok {
				continue
			}
			param := &APIParamAnnotation{}
			if v, ok := pMap["type"].(string); ok {
				param.Type = v
			}
			if v, ok := pMap["description"].(string); ok {
				param.Description = v
			}
			if v, ok := pMap["default"]; ok {
				param.Default = v
			}
			api.Query[name] = param
		}
	}

	// Responses
	if respRaw, ok := raw["responses"].(map[string]any); ok {
		api.Responses = make(map[string]*APIResponseAnnotation)
		for code, rRaw := range respRaw {
			switch r := rRaw.(type) {
			case map[string]any:
				resp := &APIResponseAnnotation{}
				if v, ok := r["description"].(string); ok {
					resp.Description = v
				}
				if contentRaw, ok := r["content"].(map[string]any); ok {
					resp.Content = make(map[string]*APIFieldAnnotation)
					for name, fRaw := range contentRaw {
						fMap, ok := fRaw.(map[string]any)
						if !ok {
							continue
						}
						resp.Content[name] = parseAPIFieldAnnotation(fMap)
					}
				}
				api.Responses[code] = resp
			}
		}
	}

	return api, nil
}

func parseAPIFieldAnnotation(raw map[string]any) *APIFieldAnnotation {
	f := &APIFieldAnnotation{}
	if v, ok := raw["type"].(string); ok {
		f.Type = v
	}
	if v, ok := raw["required"].(bool); ok {
		f.Required = v
	}
	if v, ok := raw["description"].(string); ok {
		f.Description = v
	}
	if v, ok := raw["format"].(string); ok {
		f.Format = v
	}
	if v, ok := raw["enum"].([]any); ok {
		f.Enum = toStringSlice(v)
	}
	if v, ok := raw["default"]; ok {
		f.Default = v
	}
	return f
}

func toStringSlice(arr []any) []string {
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func parseSteps(rawSteps []any) ([]Node, error) {
	if len(rawSteps) == 0 {
		return nil, nil
	}

	nodes := make([]Node, 0, len(rawSteps))
	for i, raw := range rawSteps {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, CompileError("INVALID_STEP", fmt.Sprintf("step %d must be an object", i), i)
		}

		node, err := parseStep(m, i)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func parseStep(m map[string]any, index int) (Node, error) {
	// Check for standalone comment first.
	if isCommentOnly(m) {
		node := &CommentNode{}
		node.StepIndex = index
		parseComment(&node.NodeMeta, m)
		return node, nil
	}

	// Detect step type by key presence (priority order).
	if _, ok := m["let"]; ok {
		if _, hasCall := m["call"]; hasCall {
			return parseLetCallNode(m, index)
		}
		if _, hasNew := m["new"]; hasNew {
			return parseLetNewNode(m, index)
		}
		return parseLetNode(m, index)
	}
	if _, ok := m["set"]; ok {
		return parseSetNode(m, index)
	}
	if _, ok := m["if"]; ok {
		return parseIfNode(m, index)
	}
	if _, ok := m["switch"]; ok {
		return parseSwitchNode(m, index)
	}
	if _, ok := m["for"]; ok {
		return parseForNode(m, index)
	}
	if _, ok := m["while"]; ok {
		return parseWhileNode(m, index)
	}
	if _, ok := m["return"]; ok {
		return parseReturnNode(m, index)
	}
	if _, ok := m["call"]; ok {
		return parseCallNode(m, index)
	}
	if _, ok := m["try"]; ok {
		return parseTryNode(m, index)
	}
	if _, ok := m["error"]; ok {
		return parseErrorNode(m, index)
	}
	if _, ok := m["log"]; ok {
		return parseLogNode(m, index)
	}
	if _, ok := m["parallel"]; ok {
		return parseParallelNode(m, index)
	}
	if _, ok := m["break"]; ok {
		return parseBreakNode(m, index)
	}
	if _, ok := m["continue"]; ok {
		return parseContinueNode(m, index)
	}

	// Unknown step type.
	keys := make([]string, 0, len(m))
	for k := range m {
		if k != "_c" {
			keys = append(keys, k)
		}
	}
	return nil, CompileError("UNKNOWN_STEP",
		fmt.Sprintf("unknown step type at step %d (keys: %s)", index, strings.Join(keys, ", ")), index)
}

// --- Individual step parsers ---

func parseLetNode(m map[string]any, index int) (*LetNode, error) {
	node := &LetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	name, ok := m["let"].(string)
	if !ok {
		return nil, CompileError("INVALID_LET", "let name must be a string", index)
	}
	node.Name = name

	if typ, ok := m["type"].(string); ok {
		node.Type = TypeFromJSON(typ)
	}

	modes := 0
	if _, ok := m["value"]; ok {
		node.Value = m["value"]
		node.HasValue = true
		modes++
	}
	if expr, ok := m["expr"].(string); ok {
		node.Expr = expr
		node.HasExpr = true
		modes++
	}
	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
		node.HasWith = true
		modes++
	}

	if modes == 0 {
		return nil, CompileError("MISSING_VALUE", fmt.Sprintf("let '%s' requires one of: value, expr, with", name), index)
	}
	if modes > 1 {
		return nil, CompileError("MULTIPLE_VALUES", fmt.Sprintf("let '%s' has multiple value modes (use exactly one of: value, expr, with)", name), index)
	}

	return node, nil
}

func parseLetCallNode(m map[string]any, index int) (*LetNode, error) {
	node := &LetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	name, ok := m["let"].(string)
	if !ok {
		return nil, CompileError("INVALID_LET", "let name must be a string", index)
	}
	node.Name = name

	if typ, ok := m["type"].(string); ok {
		node.Type = TypeFromJSON(typ)
	}

	callName, ok := m["call"].(string)
	if !ok {
		return nil, CompileError("INVALID_CALL", "call function name must be a string", index)
	}
	node.Call = callName

	if withRaw, ok := m["with"].(map[string]any); ok {
		node.CallWith = toStringMap(withRaw)
	}

	return node, nil
}

func parseSetNode(m map[string]any, index int) (*SetNode, error) {
	node := &SetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	target, ok := m["set"].(string)
	if !ok {
		return nil, CompileError("INVALID_SET", "set target must be a string", index)
	}
	node.Target = target

	modes := 0
	if _, ok := m["value"]; ok {
		node.Value = m["value"]
		node.HasValue = true
		modes++
	}
	if expr, ok := m["expr"].(string); ok {
		node.Expr = expr
		node.HasExpr = true
		modes++
	}
	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
		node.HasWith = true
		modes++
	}

	if modes == 0 {
		return nil, CompileError("MISSING_VALUE", fmt.Sprintf("set '%s' requires one of: value, expr, with", target), index)
	}
	if modes > 1 {
		return nil, CompileError("MULTIPLE_VALUES", fmt.Sprintf("set '%s' has multiple value modes", target), index)
	}

	return node, nil
}

func parseIfNode(m map[string]any, index int) (*IfNode, error) {
	node := &IfNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	cond, ok := m["if"].(string)
	if !ok {
		return nil, CompileError("INVALID_IF", "if condition must be a string expression", index)
	}
	node.Condition = cond

	// Parse then (required).
	thenRaw, ok := m["then"].([]any)
	if !ok {
		return nil, CompileError("MISSING_THEN", "if requires 'then' steps array", index)
	}
	thenSteps, err := parseSteps(thenRaw)
	if err != nil {
		return nil, err
	}
	node.Then = thenSteps

	// Parse elif (optional).
	if elifRaw, ok := m["elif"].([]any); ok {
		for i, eRaw := range elifRaw {
			eMap, ok := eRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] must be an object", i), index)
			}
			cond, ok := eMap["condition"].(string)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] requires 'condition' string", i), index)
			}
			thenRaw, ok := eMap["then"].([]any)
			if !ok {
				return nil, CompileError("INVALID_ELIF", fmt.Sprintf("elif[%d] requires 'then' steps array", i), index)
			}
			thenSteps, err := parseSteps(thenRaw)
			if err != nil {
				return nil, err
			}
			node.Elif = append(node.Elif, ElifBlock{Condition: cond, Then: thenSteps})
		}
	}

	// Parse else (optional).
	if elseRaw, ok := m["else"].([]any); ok {
		elseSteps, err := parseSteps(elseRaw)
		if err != nil {
			return nil, err
		}
		node.Else = elseSteps
	}

	return node, nil
}

func parseSwitchNode(m map[string]any, index int) (*SwitchNode, error) {
	node := &SwitchNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	exprStr, ok := m["switch"].(string)
	if !ok {
		return nil, CompileError("INVALID_SWITCH", "switch expression must be a string", index)
	}
	node.Expr = exprStr

	casesRaw, ok := m["cases"].(map[string]any)
	if !ok {
		return nil, CompileError("MISSING_CASES", "switch requires 'cases' object", index)
	}

	node.Cases = make(map[string][]Node)
	for key, stepsRaw := range casesRaw {
		stepsArr, ok := stepsRaw.([]any)
		if !ok {
			return nil, CompileError("INVALID_CASE", fmt.Sprintf("case '%s' must be a steps array", key), index)
		}
		steps, err := parseSteps(stepsArr)
		if err != nil {
			return nil, err
		}
		node.Cases[key] = steps
	}

	return node, nil
}

func parseForNode(m map[string]any, index int) (*ForNode, error) {
	node := &ForNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	varName, ok := m["for"].(string)
	if !ok {
		return nil, CompileError("INVALID_FOR", "for variable must be a string", index)
	}
	node.Variable = varName

	if inExpr, ok := m["in"].(string); ok {
		node.In = inExpr
	}
	if rangeRaw, ok := m["range"].([]any); ok {
		node.Range = rangeRaw
	}

	if node.In == "" && node.Range == nil {
		return nil, CompileError("MISSING_ITERABLE", "for requires 'in' expression or 'range' array", index)
	}

	if idxName, ok := m["index"].(string); ok {
		node.Index = idxName
	}

	stepsRaw, ok := m["steps"].([]any)
	if !ok {
		return nil, CompileError("MISSING_STEPS", "for requires 'steps' array", index)
	}
	steps, err := parseSteps(stepsRaw)
	if err != nil {
		return nil, err
	}
	node.Steps = steps

	return node, nil
}

func parseWhileNode(m map[string]any, index int) (*WhileNode, error) {
	node := &WhileNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	cond, ok := m["while"].(string)
	if !ok {
		return nil, CompileError("INVALID_WHILE", "while condition must be a string expression", index)
	}
	node.Condition = cond

	stepsRaw, ok := m["steps"].([]any)
	if !ok {
		return nil, CompileError("MISSING_STEPS", "while requires 'steps' array", index)
	}
	steps, err := parseSteps(stepsRaw)
	if err != nil {
		return nil, err
	}
	node.Steps = steps

	return node, nil
}

func parseReturnNode(m map[string]any, index int) (*ReturnNode, error) {
	node := &ReturnNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	retVal := m["return"]

	switch v := retVal.(type) {
	case string:
		// Expression mode (most common): {"return": "expr"}
		node.Expr = v
		node.HasExpr = true
	case map[string]any:
		if newName, ok := v["new"].(string); ok {
			node.New = newName
			node.HasNew = true
			if withRaw, ok := v["with"].(map[string]any); ok {
				node.NewWith = parseNewWithArgs(withRaw)
			}
		} else if val, ok := v["value"]; ok {
			node.Value = val
			node.HasValue = true
		} else if expr, ok := v["expr"].(string); ok {
			node.Expr = expr
			node.HasExpr = true
		} else if withRaw, ok := v["with"].(map[string]any); ok {
			node.With = toStringMap(withRaw)
			node.HasWith = true
		} else {
			node.Value = v
			node.HasValue = true
		}
	default:
		// Literal return: {"return": 42}, {"return": null}, {"return": true}
		node.Value = retVal
		node.HasValue = true
	}

	return node, nil
}

func parseCallNode(m map[string]any, index int) (*CallNode, error) {
	node := &CallNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	funcName, ok := m["call"].(string)
	if !ok {
		return nil, CompileError("INVALID_CALL", "call function name must be a string", index)
	}
	node.Function = funcName

	if withRaw, ok := m["with"].(map[string]any); ok {
		node.With = toStringMap(withRaw)
	}

	return node, nil
}

func parseTryNode(m map[string]any, index int) (*TryNode, error) {
	node := &TryNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	tryRaw, ok := m["try"].([]any)
	if !ok {
		return nil, CompileError("INVALID_TRY", "try must be a steps array", index)
	}
	trySteps, err := parseSteps(tryRaw)
	if err != nil {
		return nil, err
	}
	node.Try = trySteps

	if catchRaw, ok := m["catch"].(map[string]any); ok {
		cb := &CatchBlock{}
		if as, ok := catchRaw["as"].(string); ok {
			cb.As = as
		} else {
			cb.As = "err"
		}
		if stepsRaw, ok := catchRaw["steps"].([]any); ok {
			steps, err := parseSteps(stepsRaw)
			if err != nil {
				return nil, err
			}
			cb.Steps = steps
		}
		node.Catch = cb
	}

	if finallyRaw, ok := m["finally"].([]any); ok {
		steps, err := parseSteps(finallyRaw)
		if err != nil {
			return nil, err
		}
		node.Finally = steps
	}

	return node, nil
}

func parseErrorNode(m map[string]any, index int) (*ErrorNode, error) {
	node := &ErrorNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	errVal := m["error"]

	switch v := errVal.(type) {
	case string:
		node.Message = v
		node.IsStructured = false
	case map[string]any:
		node.IsStructured = true
		if code, ok := v["code"].(string); ok {
			node.Code = code
		}
		if msg, ok := v["message"].(string); ok {
			node.Message = msg
		}
		if details, ok := v["details"].(string); ok {
			node.Details = details
		}
	default:
		return nil, CompileError("INVALID_ERROR", "error must be a string expression or structured object", index)
	}

	return node, nil
}

func parseLogNode(m map[string]any, index int) (*LogNode, error) {
	node := &LogNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	logVal := m["log"]

	switch v := logVal.(type) {
	case string:
		node.Message = v
		node.IsStructured = false
	case map[string]any:
		node.IsStructured = true
		if msg, ok := v["message"].(string); ok {
			node.Message = msg
		}
		if level, ok := v["level"].(string); ok {
			node.Level = level
		}
		if dataRaw, ok := v["data"].(map[string]any); ok {
			node.Data = toStringMap(dataRaw)
		}
	default:
		return nil, CompileError("INVALID_LOG", "log must be a string expression or structured object", index)
	}

	return node, nil
}

func parseBreakNode(m map[string]any, index int) (*BreakNode, error) {
	node := &BreakNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)
	return node, nil
}

func parseContinueNode(m map[string]any, index int) (*ContinueNode, error) {
	node := &ContinueNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)
	return node, nil
}

// --- Function parsing ---

func parseFuncDef(name string, raw map[string]any, cleanedJSON []byte) (*FuncDef, error) {
	fd := &FuncDef{Name: name}

	if ret, ok := raw["returns"].(string); ok {
		fd.Returns = TypeFromJSON(ret)
	}

	// Parse params with preserved order.
	if paramsRaw, ok := raw["params"].(map[string]any); ok {
		orderedKeys := extractOrderedKeys(cleanedJSON, name, "params")
		if orderedKeys == nil {
			// Fallback: use map iteration order (non-deterministic but functional).
			for pName, pType := range paramsRaw {
				typStr, _ := pType.(string)
				fd.Params = append(fd.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		} else {
			for _, pName := range orderedKeys {
				pType, ok := paramsRaw[pName]
				if !ok {
					continue
				}
				typStr, _ := pType.(string)
				fd.Params = append(fd.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		}
	}

	if stepsRaw, ok := raw["steps"].([]any); ok {
		steps, err := parseSteps(stepsRaw)
		if err != nil {
			return nil, err
		}
		fd.Steps = steps
	}

	return fd, nil
}

// extractOrderedKeys parses the raw JSON to find the key order of a function's params.
// This is necessary because Go's map[string]any doesn't preserve insertion order.
func extractOrderedKeys(jsonData []byte, funcName, field string) []string {
	// Strategy: use json.Decoder to tokenize and find the params object
	// within the specific function definition.
	dec := json.NewDecoder(bytes.NewReader(jsonData))

	// Find "functions" → funcName → field
	if !seekKey(dec, "functions") {
		return nil
	}
	if !seekKey(dec, funcName) {
		return nil
	}
	if !seekKey(dec, field) {
		return nil
	}

	// Now read the keys of this object in order.
	t, err := dec.Token()
	if err != nil {
		return nil
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil
	}

	var keys []string
	depth := 0
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}

		if depth == 0 {
			if key, ok := t.(string); ok {
				keys = append(keys, key)
				// Skip the value.
				skipValue(dec)
			}
		} else {
			if delim, ok := t.(json.Delim); ok {
				switch delim {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
			}
		}
	}

	return keys
}

// seekKey advances the decoder past nested structures until it finds the given key.
func seekKey(dec *json.Decoder, key string) bool {
	depth := 0
	for {
		t, err := dec.Token()
		if err != nil {
			return false
		}

		switch v := t.(type) {
		case json.Delim:
			switch v {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth < 0 {
					return false
				}
			}
		case string:
			if depth == 1 && v == key {
				return true
			}
		}
	}
}

func skipValue(dec *json.Decoder) {
	t, err := dec.Token()
	if err != nil {
		return
	}
	if delim, ok := t.(json.Delim); ok {
		if delim == '{' || delim == '[' {
			depth := 1
			for depth > 0 {
				t, err := dec.Token()
				if err != nil {
					return
				}
				if d, ok := t.(json.Delim); ok {
					switch d {
					case '{', '[':
						depth++
					case '}', ']':
						depth--
					}
				}
			}
		}
	}
}

// --- Struct parsing ---

func parseStructDef(name string, raw map[string]any, cleanedJSON []byte) (*StructDef, error) {
	// Barrel file alias: {"alias": "_addr.Address"} → re-export
	if alias, ok := raw["alias"].(string); ok {
		return &StructDef{
			Name:   name,
			Alias:  alias,
			Fields: make(map[string]*FieldDef),
		}, nil
	}

	sd := &StructDef{
		Name:   name,
		Fields: make(map[string]*FieldDef),
	}

	if frozen, ok := raw["frozen"].(bool); ok {
		sd.Frozen = frozen
	}

	fieldsRaw, ok := raw["fields"].(map[string]any)
	if !ok {
		return nil, CompileError("INVALID_STRUCT", fmt.Sprintf("struct '%s' requires 'fields' object", name), -1)
	}

	for fieldName, fieldRaw := range fieldsRaw {
		fd, err := parseFieldDef(name, fieldName, fieldRaw)
		if err != nil {
			return nil, err
		}
		sd.Fields[fieldName] = fd
	}

	if methodsRaw, ok := raw["methods"].(map[string]any); ok {
		sd.Methods = make(map[string]*MethodDef)
		for methodName, mRaw := range methodsRaw {
			mMap, ok := mRaw.(map[string]any)
			if !ok {
				return nil, CompileError("INVALID_METHOD",
					fmt.Sprintf("method '%s.%s' must be an object", name, methodName), -1)
			}
			md, err := parseMethodDef(name, methodName, mMap, cleanedJSON)
			if err != nil {
				return nil, err
			}
			sd.Methods[methodName] = md
		}
	}

	return sd, nil
}

func parseFieldDef(structName, fieldName string, raw any) (*FieldDef, error) {
	switch v := raw.(type) {
	case string:
		return &FieldDef{Type: TypeFromJSON(v)}, nil
	case map[string]any:
		fd := &FieldDef{}
		typStr, ok := v["type"].(string)
		if !ok {
			return nil, CompileError("INVALID_FIELD",
				fmt.Sprintf("field '%s.%s' requires 'type' string", structName, fieldName), -1)
		}
		fd.Type = TypeFromJSON(typStr)
		if def, ok := v["default"]; ok {
			fd.Default = def
			fd.HasDefault = true
		}
		return fd, nil
	default:
		return nil, CompileError("INVALID_FIELD",
			fmt.Sprintf("field '%s.%s' must be a type string or {type, default} object", structName, fieldName), -1)
	}
}

func parseMethodDef(structName, methodName string, raw map[string]any, cleanedJSON []byte) (*MethodDef, error) {
	md := &MethodDef{Name: methodName}

	if ret, ok := raw["returns"].(string); ok {
		md.Returns = TypeFromJSON(ret)
	}

	if paramsRaw, ok := raw["params"].(map[string]any); ok {
		orderedKeys := extractMethodParamKeys(cleanedJSON, structName, methodName)
		if orderedKeys == nil {
			for pName, pType := range paramsRaw {
				typStr, _ := pType.(string)
				md.Params = append(md.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		} else {
			for _, pName := range orderedKeys {
				pType, ok := paramsRaw[pName]
				if !ok {
					continue
				}
				typStr, _ := pType.(string)
				md.Params = append(md.Params, FuncParam{
					Name: pName,
					Type: TypeFromJSON(typStr),
				})
			}
		}
	}

	if stepsRaw, ok := raw["steps"].([]any); ok {
		steps, err := parseSteps(stepsRaw)
		if err != nil {
			return nil, err
		}
		md.Steps = steps
	}

	return md, nil
}

func extractMethodParamKeys(jsonData []byte, structName, methodName string) []string {
	dec := json.NewDecoder(bytes.NewReader(jsonData))
	if !seekKey(dec, "structs") {
		return nil
	}
	if !seekKey(dec, structName) {
		return nil
	}
	if !seekKey(dec, "methods") {
		return nil
	}
	if !seekKey(dec, methodName) {
		return nil
	}
	if !seekKey(dec, "params") {
		return nil
	}

	t, err := dec.Token()
	if err != nil {
		return nil
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil
	}

	var keys []string
	depth := 0
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			break
		}
		if depth == 0 {
			if key, ok := t.(string); ok {
				keys = append(keys, key)
				skipValue(dec)
			}
		} else {
			if delim, ok := t.(json.Delim); ok {
				switch delim {
				case '{', '[':
					depth++
				case '}', ']':
					depth--
				}
			}
		}
	}
	return keys
}

func parseLetNewNode(m map[string]any, index int) (*LetNode, error) {
	node := &LetNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	name, ok := m["let"].(string)
	if !ok {
		return nil, CompileError("INVALID_LET", "let name must be a string", index)
	}
	node.Name = name

	if typ, ok := m["type"].(string); ok {
		node.Type = TypeFromJSON(typ)
	}

	newName, ok := m["new"].(string)
	if !ok {
		return nil, CompileError("INVALID_NEW", "new struct name must be a string", index)
	}
	node.New = newName

	if withRaw, ok := m["with"].(map[string]any); ok {
		node.NewWith = parseNewWithArgs(withRaw)
	}

	return node, nil
}

// parseNewWithArgs handles mixed value types in new+with:
//   - string → expression (kept as string)
//   - number/bool/null → literal expression string
//   - map with "new" key → nested NewConstruction{StructName, With}
//   - map without "new" → literal value (kept as-is)
//   - array → literal value (kept as-is)
func parseNewWithArgs(m map[string]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = val
		case map[string]any:
			if newName, ok := val["new"].(string); ok {
				nc := &NewConstruction{StructName: newName}
				if withRaw, ok := val["with"].(map[string]any); ok {
					nc.With = parseNewWithArgs(withRaw)
				}
				result[k] = nc
			} else {
				result[k] = val
			}
		case nil:
			result[k] = "nil"
		case bool:
			if val {
				result[k] = "true"
			} else {
				result[k] = "false"
			}
		default:
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// --- Parallel parsing ---

func parseParallelNode(m map[string]any, index int) (*ParallelNode, error) {
	node := &ParallelNode{}
	node.StepIndex = index
	parseComment(&node.NodeMeta, m)

	branchesRaw, ok := m["parallel"].(map[string]any)
	if !ok {
		return nil, CompileError("INVALID_PARALLEL", "parallel must be an object of branch_name → steps", index)
	}

	node.Branches = make(map[string][]Node)
	for branchName, stepsRaw := range branchesRaw {
		stepsArr, ok := stepsRaw.([]any)
		if !ok {
			return nil, CompileError("INVALID_PARALLEL",
				fmt.Sprintf("parallel branch '%s' must be a steps array", branchName), index)
		}
		steps, err := parseSteps(stepsArr)
		if err != nil {
			return nil, err
		}
		node.Branches[branchName] = steps
	}

	if join, ok := m["join"].(string); ok {
		node.Join = join
	}
	if onError, ok := m["on_error"].(string); ok {
		node.OnError = onError
	}
	if into, ok := m["into"].(string); ok {
		node.Into = into
	}

	return node, nil
}

// --- Import helpers ---

func detectImportPathType(path string) string {
	if strings.HasPrefix(path, "stdlib:") {
		return "stdlib"
	}
	if strings.HasPrefix(path, "ext:") {
		return "ext"
	}
	if strings.HasPrefix(path, "io:") {
		return "io"
	}
	return "relative"
}

// --- Helpers ---

func parseComment(meta *NodeMeta, m map[string]any) {
	c, ok := m["_c"]
	if !ok {
		return
	}
	switch v := c.(type) {
	case string:
		meta.Comment = v
	case []any:
		for _, line := range v {
			if s, ok := line.(string); ok {
				meta.Comments = append(meta.Comments, s)
			}
		}
	}
}

func isCommentOnly(m map[string]any) bool {
	for k := range m {
		if k != "_c" {
			return false
		}
	}
	_, hasComment := m["_c"]
	return hasComment
}

func parseLimits(raw map[string]any) *LimitsDef {
	ld := &LimitsDef{}
	if v, ok := toInt(raw["max_depth"]); ok {
		ld.MaxDepth = &v
	}
	if v, ok := toInt(raw["max_steps"]); ok {
		ld.MaxSteps = &v
	}
	if v, ok := toInt(raw["max_loop_iterations"]); ok {
		ld.MaxLoopIterations = &v
	}
	if v, ok := toInt(raw["max_nodes"]); ok {
		ld.MaxNodes = &v
	}
	if v, ok := toInt(raw["max_variables"]); ok {
		ld.MaxVariables = &v
	}
	if v, ok := toInt(raw["max_variable_size"]); ok {
		ld.MaxVariableSize = &v
	}
	if v, ok := toInt(raw["max_output_size"]); ok {
		ld.MaxOutputSize = &v
	}
	if v, ok := raw["timeout"].(string); ok {
		ld.Timeout = &v
	}
	return ld
}

func toStringMap(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		} else {
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// toExprMap converts JSON values to expression strings.
// string → kept as expression, number/bool → literal expression, null → "nil"
func toExprMap(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = val
		case nil:
			result[k] = "nil"
		case bool:
			if val {
				result[k] = "true"
			} else {
				result[k] = "false"
			}
		default:
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	}
	return 0, false
}
