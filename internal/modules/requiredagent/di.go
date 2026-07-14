package requiredagent

import (
	"github.com/samber/do/v2"
	"github.com/svetlyopet/heimdallr/internal/logger"
	"gorm.io/gorm"
)

var Package = do.Package(
	do.Lazy(provideRepository),
	do.Lazy(provideService),
	do.Lazy(provideHandler),
)

func provideRepository(i do.Injector) (Repository, error) {
	return NewRepository(do.MustInvoke[*gorm.DB](i)), nil
}

func provideService(i do.Injector) (Service, error) {
	return NewService(
		do.MustInvoke[Repository](i),
		do.MustInvoke[*logger.Logger](i),
	), nil
}

func provideHandler(i do.Injector) (Handler, error) {
	return NewHandler(do.MustInvoke[Service](i))
}
