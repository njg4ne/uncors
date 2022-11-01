package mock

import (
	"github.com/evg4b/uncors/internal/contracts"
	"github.com/gorilla/mux"
)

func MakeMockedRoutes(router *mux.Router, logger contracts.Logger, mocks []Mock) {
	var defaultMocks []Mock

	for _, mock := range mocks {
		if len(mock.Queries) > 0 || len(mock.Headers) > 0 || len(mock.Method) > 0 {
			route := router.NewRoute()

			setPath(route, mock.Path)
			setMethod(route, mock.Method)
			setQueries(route, mock.Queries)
			setHeaders(route, mock.Headers)

			handler := NewMockHandler(WithResponse(mock.Response), WithLogger(logger))
			route.Handler(handler)
		} else {
			defaultMocks = append(defaultMocks, mock)
		}
	}

	for _, mock := range defaultMocks {
		route := router.NewRoute()

		setPath(route, mock.Path)

		handler := NewMockHandler(WithResponse(mock.Response), WithLogger(logger))
		route.Handler(handler)
	}
}

func setPath(route *mux.Route, path string) {
	if len(path) > 0 {
		route.Path(path)
	}
}

func setMethod(route *mux.Route, methods string) {
	if len(methods) > 0 {
		route.Methods(methods)
	}
}

func setQueries(route *mux.Route, queries map[string]string) {
	if len(queries) > 0 {
		for key, value := range queries {
			route.Queries(key, value)
		}
	}
}

func setHeaders(route *mux.Route, headers map[string]string) {
	if len(headers) > 0 {
		for key, value := range headers {
			route.Headers(key, value)
		}
	}
}
