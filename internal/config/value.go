package config

import "github.com/samber/do/v2"

func ValuePackage(cfg *AppConfig) func(do.Injector) {
	return func(i do.Injector) {
		do.ProvideValue(i, cfg)
	}
}
