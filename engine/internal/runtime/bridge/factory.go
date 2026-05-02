package bridge

import (
	"github.com/bitcode-framework/bitcode/internal/domain/event"
	"github.com/bitcode-framework/bitcode/internal/domain/model"
	domainstorage "github.com/bitcode-framework/bitcode/internal/domain/storage"
	infracache "github.com/bitcode-framework/bitcode/internal/infrastructure/cache"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/i18n"
	"github.com/bitcode-framework/bitcode/internal/infrastructure/persistence"
	"github.com/bitcode-framework/bitcode/internal/runtime/executor"
	"github.com/bitcode-framework/bitcode/internal/presentation/websocket"
	"github.com/bitcode-framework/bitcode/pkg/email"
	"github.com/bitcode-framework/bitcode/pkg/security"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Factory struct {
	DB              *gorm.DB
	ModelRegistry   *model.Registry
	PermService     *persistence.PermissionService
	Cache           infracache.Cache
	EventBus        *event.Bus
	Config          *viper.Viper
	EmailSender     email.Sender
	WSHub           *websocket.Hub
	StorageDriver   domainstorage.StorageDriver
	Translator      *i18n.Translator
	AuditRepo       *persistence.AuditLogRepository
	Encryptor       *security.FieldEncryptor
	Executor        *executor.Executor
	ProcessRegistry ProcessRegistry
	ExecLogConfig   ExecutionLogConfig
	Meta            MetaProvider
	Refresher       ModelRefresher
}

func (f *Factory) NewContext(moduleName string, session Session, rules SecurityRules) *Context {
	txb := newTxBridge(f.DB)
	return &Context{
		txManager: newTxManager(f.DB),
		txBridge:  txb,
		model:     newModelFactoryWithTx(f.DB, f.ModelRegistry, f.PermService, txb),
		db:        newDBBridgeWithTx(f.DB, txb),
		http:      newHTTPBridge(),
		cache:     newCacheBridge(f.Cache),
		fs:        newFSBridge(moduleName, rules),
		session:   session,
		config:    newConfigBridge(f.Config, moduleName),
		env:       newEnvBridge(f.Config, rules, moduleName),
		emitter:   newEventBridge(f.EventBus),
		caller:    newProcessBridge(f.Executor, f.ProcessRegistry, session),
		execer:    newExecBridge(rules),
		logger:    newLogBridge(moduleName),
		email:     newEmailBridge(f.EmailSender),
		notify:    newNotifyBridge(f.WSHub, session),
		storage:   newStorageBridge(f.StorageDriver),
		i18n:      newI18NBridge(f.Translator),
		security:  newSecurityBridge(f.PermService, session),
		audit:     newAuditBridge(f.AuditRepo, session, moduleName),
		crypto:    newCryptoBridge(f.Encryptor),
		execution: newExecutionBridge(f.DB, session),
		meta:      f.Meta,
		refresher: f.Refresher,
	}
}
