package rbac

import "github.com/samber/do/v2"

var Package = do.Package(
	do.Lazy(provideAuthorizer),
)

func provideAuthorizer(_ do.Injector) (Authorizer, error) {
	return NewAuthorizer(), nil
}
