package io

// All returns all I/O modules with the given security config.
func All(security *SecurityConfig) []IOModule {
	if security == nil {
		security = DefaultSecurityConfig()
	}
	return []IOModule{
		NewHTTPModule(security),
		NewFSModule(security),
		NewSQLModule(security),
		NewExecModule(security),
		NewMongoModule(security),
		NewRedisModule(security),
	}
}

// HTTP returns just the HTTP module.
func HTTP(security *SecurityConfig) IOModule {
	return NewHTTPModule(security)
}

// FS returns just the FS module.
func FS(security *SecurityConfig) IOModule {
	return NewFSModule(security)
}

// SQL returns just the SQL module.
func SQL(security *SecurityConfig) IOModule {
	return NewSQLModule(security)
}

// Exec returns just the Exec module.
func Exec(security *SecurityConfig) IOModule {
	return NewExecModule(security)
}

// Mongo returns just the Mongo module.
func Mongo(security *SecurityConfig) IOModule {
	return NewMongoModule(security)
}

// Redis returns just the Redis module.
func Redis(security *SecurityConfig) IOModule {
	return NewRedisModule(security)
}
