package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
	"github.com/bitcode-framework/bitcode/internal/runtime/refresh"
)

type MetaHandler struct {
	modelRegistry   *domainModel.Registry
	moduleRegistry  *module.Registry
	processRegistry *executor.ProcessRegistry
	viewResolver    func(name string) *parser.ViewDefinition
	viewLister      func() map[string]*parser.ViewDefinition
	refreshSched    *refresh.Scheduler
	syncFn          func(modelName string) error
}

type MetaHandlerConfig struct {
	ModelRegistry   *domainModel.Registry
	ModuleRegistry  *module.Registry
	ProcessRegistry *executor.ProcessRegistry
	ViewResolver    func(name string) *parser.ViewDefinition
	ViewLister      func() map[string]*parser.ViewDefinition
	RefreshSched    *refresh.Scheduler
	SyncFn          func(modelName string) error
}

func NewMetaHandler(cfg MetaHandlerConfig) *MetaHandler {
	return &MetaHandler{
		modelRegistry:   cfg.ModelRegistry,
		moduleRegistry:  cfg.ModuleRegistry,
		processRegistry: cfg.ProcessRegistry,
		viewResolver:    cfg.ViewResolver,
		viewLister:      cfg.ViewLister,
		refreshSched:    cfg.RefreshSched,
		syncFn:          cfg.SyncFn,
	}
}

func (h *MetaHandler) RegisterRoutes(app *fiber.App, authMiddleware ...fiber.Handler) {
	meta := app.Group("/api/v1/_meta")
	for _, mw := range authMiddleware {
		meta.Use(mw)
	}

	meta.Get("/models", h.ListModels)
	meta.Get("/models/:name", h.GetModel)
	meta.Get("/models/:name/fields", h.GetModelFields)
	meta.Get("/views", h.ListViews)
	meta.Get("/views/:name", h.GetView)
	meta.Get("/modules", h.ListModules)
	meta.Get("/modules/:name", h.GetModule)
	meta.Get("/processes", h.ListProcesses)
	meta.Get("/processes/:name", h.GetProcess)
	meta.Get("/field-types", h.ListFieldTypes)
	meta.Post("/models/:name/refresh", h.RefreshModel)
}

func (h *MetaHandler) ListModels(c *fiber.Ctx) error {
	models := h.modelRegistry.List()
	result := make([]fiber.Map, 0, len(models))
	for _, m := range models {
		tableName := h.modelRegistry.TableName(m.Name)
		result = append(result, fiber.Map{
			"name":        m.Name,
			"module":      m.Module,
			"label":       m.Label,
			"title_field": m.TitleField,
			"field_count": len(m.Fields),
			"table_name":  tableName,
			"source":      m.GetEffectiveSource(),
			"has_api":     m.API != nil,
		})
	}
	return c.JSON(fiber.Map{"models": result})
}

func (h *MetaHandler) GetModel(c *fiber.Ctx) error {
	name := c.Params("name")
	m, err := h.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	fields := make(map[string]fiber.Map, len(m.Fields))
	for fname, fd := range m.Fields {
		fmap := fiber.Map{
			"type": string(fd.Type),
		}
		if fd.Label != "" {
			fmap["label"] = fd.Label
		}
		if fd.Required {
			fmap["required"] = true
		}
		if fd.Unique {
			fmap["unique"] = true
		}
		if fd.Max > 0 {
			fmap["max"] = fd.Max
		}
		if fd.Model != "" {
			fmap["model"] = fd.Model
		}
		if fd.Inverse != "" {
			fmap["inverse"] = fd.Inverse
		}
		if len(fd.Options) > 0 {
			fmap["options"] = fd.Options
		}
		if fd.Hidden {
			fmap["hidden"] = true
		}
		if fd.Default != nil {
			fmap["default"] = fd.Default
		}
		fields[fname] = fmap
	}

	result := fiber.Map{
		"name":         m.Name,
		"module":       m.Module,
		"label":        m.Label,
		"title_field":  m.TitleField,
		"search_field": m.SearchField,
		"table_name":   h.modelRegistry.TableName(m.Name),
		"fields":       fields,
		"source":       m.GetEffectiveSource(),
		"timestamps":   m.IsTimestamps(),
		"soft_deletes": m.IsSoftDeletes(),
	}
	if m.PrimaryKey != nil {
		result["primary_key"] = m.PrimaryKey
	}
	if len(m.Indexes) > 0 {
		result["indexes"] = m.Indexes
	}
	if len(m.RecordRules) > 0 {
		result["record_rules"] = m.RecordRules
	}
	if m.API != nil {
		result["api"] = fiber.Map{
			"auto_crud": m.API.AutoCRUD,
			"auth":      m.API.Auth,
		}
	}
	return c.JSON(result)
}

func (h *MetaHandler) GetModelFields(c *fiber.Ctx) error {
	name := c.Params("name")
	m, err := h.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}

	fields := make(map[string]fiber.Map, len(m.Fields))
	for fname, fd := range m.Fields {
		fmap := fiber.Map{"type": string(fd.Type)}
		if fd.Label != "" {
			fmap["label"] = fd.Label
		}
		if fd.Required {
			fmap["required"] = true
		}
		if fd.Unique {
			fmap["unique"] = true
		}
		if fd.Max > 0 {
			fmap["max"] = fd.Max
		}
		if fd.Model != "" {
			fmap["model"] = fd.Model
		}
		if len(fd.Options) > 0 {
			fmap["options"] = fd.Options
		}
		if fd.Hidden {
			fmap["hidden"] = true
		}
		fields[fname] = fmap
	}
	return c.JSON(fiber.Map{"fields": fields})
}

func (h *MetaHandler) ListViews(c *fiber.Ctx) error {
	if h.viewLister == nil {
		return c.JSON(fiber.Map{"views": []any{}})
	}
	views := h.viewLister()
	result := make([]fiber.Map, 0, len(views))
	for name, v := range views {
		result = append(result, fiber.Map{
			"name":  name,
			"type":  string(v.Type),
			"model": v.Model,
			"title": v.Title,
		})
	}
	return c.JSON(fiber.Map{"views": result})
}

func (h *MetaHandler) GetView(c *fiber.Ctx) error {
	name := c.Params("name")
	if h.viewResolver == nil {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}
	v := h.viewResolver(name)
	if v == nil {
		return c.Status(404).JSON(fiber.Map{"error": "view not found"})
	}
	return c.JSON(v)
}

func (h *MetaHandler) ListModules(c *fiber.Ctx) error {
	names := h.moduleRegistry.InstalledNames()
	result := make([]fiber.Map, 0, len(names))
	for _, name := range names {
		installed, err := h.moduleRegistry.Get(name)
		if err != nil {
			continue
		}
		result = append(result, fiber.Map{
			"name":    installed.Definition.Name,
			"label":   installed.Definition.Label,
			"version": installed.Definition.Version,
			"depends": installed.Definition.Depends,
		})
	}
	return c.JSON(fiber.Map{"modules": result})
}

func (h *MetaHandler) GetModule(c *fiber.Ctx) error {
	name := c.Params("name")
	installed, err := h.moduleRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "module not found"})
	}
	return c.JSON(installed.Definition)
}

func (h *MetaHandler) ListProcesses(c *fiber.Ctx) error {
	names := h.processRegistry.List()
	result := make([]fiber.Map, 0, len(names))
	for _, name := range names {
		result = append(result, fiber.Map{"name": name})
	}
	return c.JSON(fiber.Map{"processes": result})
}

func (h *MetaHandler) GetProcess(c *fiber.Ctx) error {
	name := c.Params("name")
	proc, err := h.processRegistry.LoadProcess(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "process not found"})
	}
	return c.JSON(proc)
}

func (h *MetaHandler) ListFieldTypes(c *fiber.Ctx) error {
	types := []fiber.Map{
		{"name": "string", "category": "core", "storage": "VARCHAR(255)", "widget": "bc-field-string"},
		{"name": "text", "category": "core", "storage": "TEXT", "widget": "bc-field-text"},
		{"name": "integer", "category": "number", "storage": "INTEGER", "widget": "bc-field-integer"},
		{"name": "decimal", "category": "number", "storage": "NUMERIC(18,4)", "widget": "bc-field-decimal"},
		{"name": "float", "category": "number", "storage": "REAL", "widget": "bc-field-float"},
		{"name": "currency", "category": "number", "storage": "NUMERIC(18,2)", "widget": "bc-field-currency"},
		{"name": "percent", "category": "number", "storage": "NUMERIC(5,2)", "widget": "bc-field-percent"},
		{"name": "boolean", "category": "core", "storage": "BOOLEAN", "widget": "bc-field-checkbox"},
		{"name": "toggle", "category": "core", "storage": "BOOLEAN", "widget": "bc-field-toggle"},
		{"name": "date", "category": "datetime", "storage": "DATE", "widget": "bc-field-date"},
		{"name": "time", "category": "datetime", "storage": "TIME", "widget": "bc-field-time"},
		{"name": "datetime", "category": "datetime", "storage": "TIMESTAMP", "widget": "bc-field-datetime"},
		{"name": "duration", "category": "datetime", "storage": "VARCHAR(20)", "widget": "bc-field-duration"},
		{"name": "year", "category": "datetime", "storage": "SMALLINT", "widget": "bc-field-number"},
		{"name": "selection", "category": "choice", "storage": "VARCHAR(100)", "widget": "bc-field-select"},
		{"name": "radio", "category": "choice", "storage": "VARCHAR(100)", "widget": "bc-field-radio"},
		{"name": "email", "category": "core", "storage": "VARCHAR(255)", "widget": "bc-field-string"},
		{"name": "password", "category": "core", "storage": "VARCHAR(255)", "widget": "bc-field-password"},
		{"name": "many2one", "category": "relation", "storage": "VARCHAR(36)", "widget": "bc-field-link"},
		{"name": "one2many", "category": "relation", "storage": "virtual", "widget": ""},
		{"name": "many2many", "category": "relation", "storage": "junction_table", "widget": "bc-field-tags"},
		{"name": "morph_to", "category": "relation", "storage": "VARCHAR(36)+VARCHAR(100)", "widget": "bc-field-morph"},
		{"name": "morph_one", "category": "relation", "storage": "virtual", "widget": "bc-field-link"},
		{"name": "morph_many", "category": "relation", "storage": "virtual", "widget": ""},
		{"name": "morph_to_many", "category": "relation", "storage": "junction_table", "widget": "bc-field-tags"},
		{"name": "morph_by_many", "category": "relation", "storage": "virtual", "widget": ""},
		{"name": "json", "category": "special", "storage": "TEXT/JSONB", "widget": "bc-field-json"},
		{"name": "file", "category": "media", "storage": "VARCHAR(500)", "widget": "bc-field-file"},
		{"name": "image", "category": "media", "storage": "VARCHAR(500)", "widget": "bc-field-image"},
		{"name": "uuid", "category": "special", "storage": "VARCHAR(36)", "widget": "bc-field-string"},
		{"name": "ip", "category": "special", "storage": "VARCHAR(45)", "widget": "bc-field-string"},
		{"name": "vector", "category": "special", "storage": "vector(N)", "widget": "", "default_hidden": true},
		{"name": "binary", "category": "special", "storage": "BLOB", "widget": "", "default_hidden": true},
		{"name": "color", "category": "ui", "storage": "VARCHAR(7)", "widget": "bc-field-color"},
		{"name": "rating", "category": "ui", "storage": "INTEGER", "widget": "bc-field-rating"},
		{"name": "geolocation", "category": "special", "storage": "VARCHAR(50)", "widget": "bc-field-geo"},
		{"name": "barcode", "category": "special", "storage": "VARCHAR(255)", "widget": "bc-field-barcode"},
		{"name": "signature", "category": "media", "storage": "TEXT", "widget": "bc-field-signature"},
		{"name": "computed", "category": "special", "storage": "virtual", "widget": ""},
	}
	return c.JSON(fiber.Map{"types": types})
}

func (h *MetaHandler) RefreshModel(c *fiber.Ctx) error {
	name := c.Params("name")
	m, err := h.modelRegistry.Get(name)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "model not found"})
	}
	if !m.IsArraySource() && !m.IsProcessSource() {
		return c.Status(400).JSON(fiber.Map{"error": "only array/process source models can be refreshed"})
	}

	if h.syncFn != nil {
		if err := h.syncFn(name); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.JSON(fiber.Map{"status": "ok", "model": name})
}
