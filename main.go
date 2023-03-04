package main

import (
	"fmt"
	"net/http"

	routing "github.com/go-ozzo/ozzo-routing"
	"github.com/go-ozzo/ozzo-routing/content"
	v1 "github.com/sep/anthology/api/v1"
	"github.com/sep/anthology/app"
	"github.com/sep/anthology/registry"
	"github.com/sep/anthology/services"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := app.LoadConfig(); err != nil {
		panic(fmt.Errorf("invalid configuration: %s", err))
	}

	logger := logrus.New()

	var r registry.Registry

	switch app.Config.Backend {
	case "blob":
		r = registry.NewBlobRegistry(app.Config.Blob)
		break
	case "s3":
		r = registry.NewS3Registry(app.Config.S3)
		break
	case "filesystem":
		r = registry.NewFilesystemRegistry(app.Config.FileSystem)
		break
	}
	http.Handle("/", buildRouter(logger, r))

	address := fmt.Sprintf(":%v", app.Config.Port)
	logger.Infof("server %v is started at %v", app.Version, address)

	if app.Config.SSLConfig.IsValid() {
		panic(http.ListenAndServeTLS(address, app.Config.SSLConfig.Certificate, app.Config.SSLConfig.Key, nil))
	} else {
		panic(http.ListenAndServe(address, nil))
	}
}

func buildRouter(logger *logrus.Logger, reg registry.Registry) *routing.Router {
	router := routing.New()

	router.To("GET,HEAD", "/ping", func(c *routing.Context) error {
		c.Abort()
		return c.Write("OK" + app.Version)
	})

	router.Use(
		app.Init(logger),
		content.TypeNegotiator(content.JSON),
	)

	router.To("GET", "/.well-known/terraform.json", func(c *routing.Context) error {
		c.Abort()
		return c.Write(map[string]string{
			"modules.v1": "/v1/modules/",
		})
	})

	rg := router.Group("/v1/modules")

	v1.ServeModuleResource(rg, services.NewModuleService(reg))
	return router
}
