package bridge

import (
	"fmt"

	"github.com/bitcode-framework/bitcode/internal/compiler/parser"
	domainModel "github.com/bitcode-framework/bitcode/internal/domain/model"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/module"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
)

type metaBridge struct {
	modelRegistry   *domainModel.Registry
	moduleRegistry  *module.Registry
	processRegistry *executor.ProcessRegistry
	viewResolver    func(name string) *parser.ViewDefinition
	viewLister      func() []map[string]any
}

type MetaBridgeConfig struct {
	ModelRegistry   *domainModel.Registry
	ModuleRegistry  *module.Registry
	ProcessRegistry *executor.ProcessRegistry
	ViewResolver    func(name string) *parser.ViewDefinition
	ViewLister      func() []map[string]any
}

func NewMetaBridge(cfg MetaBridgeConfig) MetaProvider {
	return &metaBridge{
		modelRegistry:   cfg.ModelRegistry,
		moduleRegistry:  cfg.ModuleRegistry,
		processRegistry: cfg.ProcessRegistry,
		viewResolver:    cfg.ViewResolver,
		viewLister:      cfg.ViewLister,
	}
}

func (m *metaBridge) Models() ([]map[string]any, error) {
	models := m.modelRegistry.List()
	result := make([]map[string]any, 0, len(models))
	for _, md := range models {
		result = append(result, map[string]any{
			"name":        md.Name,
			"module":      md.Module,
			"label":       md.Label,
			"title_field": md.TitleField,
			"field_count": len(md.Fields),
			"table_name":  m.modelRegistry.TableName(md.Name),
			"source":      md.GetEffectiveSource(),
		})
	}
	return result, nil
}

func (m *metaBridge) Model(name string) (map[string]any, error) {
	md, err := m.modelRegistry.Get(name)
	if err != nil {
		return nil, fmt.Errorf("model %q not found", name)
	}

	fields := make(map[string]any, len(md.Fields))
	for fname, fd := range md.Fields {
		fmap := map[string]any{"type": string(fd.Type)}
		if fd.Label != "" {
			fmap["label"] = fd.Label
		}
		if fd.Required {
			fmap["required"] = true
		}
		if fd.Model != "" {
			fmap["model"] = fd.Model
		}
		if len(fd.Options) > 0 {
			fmap["options"] = fd.Options
		}
		fields[fname] = fmap
	}

	return map[string]any{
		"name":         md.Name,
		"module":       md.Module,
		"label":        md.Label,
		"title_field":  md.TitleField,
		"search_field": md.SearchField,
		"table_name":   m.modelRegistry.TableName(md.Name),
		"fields":       fields,
		"source":       md.GetEffectiveSource(),
		"timestamps":   md.IsTimestamps(),
		"soft_deletes": md.IsSoftDeletes(),
	}, nil
}

func (m *metaBridge) Fields(modelName string) (map[string]any, error) {
	md, err := m.modelRegistry.Get(modelName)
	if err != nil {
		return nil, fmt.Errorf("model %q not found", modelName)
	}

	fields := make(map[string]any, len(md.Fields))
	for fname, fd := range md.Fields {
		fmap := map[string]any{"type": string(fd.Type)}
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
		fields[fname] = fmap
	}
	return fields, nil
}

func (m *metaBridge) FieldTypes() ([]map[string]any, error) {
	types := []map[string]any{
		{"name": "string", "category": "core", "widget": "bc-field-string"},
		{"name": "text", "category": "core", "widget": "bc-field-text"},
		{"name": "integer", "category": "number", "widget": "bc-field-integer"},
		{"name": "decimal", "category": "number", "widget": "bc-field-decimal"},
		{"name": "float", "category": "number", "widget": "bc-field-float"},
		{"name": "currency", "category": "number", "widget": "bc-field-currency"},
		{"name": "boolean", "category": "core", "widget": "bc-field-checkbox"},
		{"name": "date", "category": "datetime", "widget": "bc-field-date"},
		{"name": "datetime", "category": "datetime", "widget": "bc-field-datetime"},
		{"name": "selection", "category": "choice", "widget": "bc-field-select"},
		{"name": "email", "category": "core", "widget": "bc-field-string"},
		{"name": "many2one", "category": "relation", "widget": "bc-field-link"},
		{"name": "one2many", "category": "relation", "widget": ""},
		{"name": "many2many", "category": "relation", "widget": "bc-field-tags"},
		{"name": "json", "category": "special", "widget": "bc-field-json"},
		{"name": "file", "category": "media", "widget": "bc-field-file"},
		{"name": "image", "category": "media", "widget": "bc-field-image"},
	}
	return types, nil
}

func (m *metaBridge) Views() ([]map[string]any, error) {
	if m.viewLister == nil {
		return []map[string]any{}, nil
	}
	return m.viewLister(), nil
}

func (m *metaBridge) View(name string) (map[string]any, error) {
	if m.viewResolver == nil {
		return nil, fmt.Errorf("view %q not found", name)
	}
	v := m.viewResolver(name)
	if v == nil {
		return nil, fmt.Errorf("view %q not found", name)
	}
	return map[string]any{
		"name":   v.Name,
		"type":   string(v.Type),
		"model":  v.Model,
		"title":  v.Title,
		"fields": v.Fields,
	}, nil
}

func (m *metaBridge) Modules() ([]map[string]any, error) {
	names := m.moduleRegistry.InstalledNames()
	result := make([]map[string]any, 0, len(names))
	for _, name := range names {
		installed, err := m.moduleRegistry.Get(name)
		if err != nil {
			continue
		}
		result = append(result, map[string]any{
			"name":    installed.Definition.Name,
			"label":   installed.Definition.Label,
			"version": installed.Definition.Version,
		})
	}
	return result, nil
}

func (m *metaBridge) Module(name string) (map[string]any, error) {
	installed, err := m.moduleRegistry.Get(name)
	if err != nil {
		return nil, fmt.Errorf("module %q not found", name)
	}
	return map[string]any{
		"name":    installed.Definition.Name,
		"label":   installed.Definition.Label,
		"version": installed.Definition.Version,
		"depends": installed.Definition.Depends,
	}, nil
}

func (m *metaBridge) Processes() ([]map[string]any, error) {
	names := m.processRegistry.List()
	result := make([]map[string]any, 0, len(names))
	for _, name := range names {
		result = append(result, map[string]any{"name": name})
	}
	return result, nil
}
