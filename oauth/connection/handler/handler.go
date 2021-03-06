// Package handler
//
// Defined permissions:
// * permission("rn:hydra:oauth2:connections") actions("create", "get")
// * permission("rn:hydra:oauth2:connections:%s", id) actions("get", "delete")
package handler

import (
	"encoding/json"
	"fmt"
	"github.com/asaskevich/govalidator"
	"github.com/go-errors/errors"
	"github.com/gorilla/mux"
	hydcon "github.com/ory-am/common/handler"
	. "github.com/ory-am/common/pkg"
	"github.com/ory-am/hydra/middleware"
	. "github.com/ory-am/hydra/oauth/connection"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"net/http"
)

var connectionsPermission = "rn:hydra:oauth2:connections"

func permission(id string) string {
	return fmt.Sprintf("rn:hydra:oauth2:connections:%s", id)
}

type Handler struct {
	s Storage
	m middleware.Middleware
}

type payload struct {
	ID string `json:"id,omitempty" `
}

func NewHandler(s Storage, m middleware.Middleware) *Handler {
	return &Handler{s, m}
}

func (h *Handler) SetRoutes(r *mux.Router, extractor func(h hydcon.ContextHandler) hydcon.ContextHandler) {
	r.Handle("/oauth2/connections", hydcon.NewContextAdapter(
		context.Background(),
		extractor,
		h.m.IsAuthenticated,
		h.m.IsAuthorized(connectionsPermission, "create", nil),
	).ThenFunc(h.Create)).Methods("POST")

	r.Handle("/oauth2/connections", hydcon.NewContextAdapter(
		context.Background(),
		extractor,
		h.m.IsAuthenticated,
	).ThenFunc(h.Find)).Queries("subject", "{subject}").Methods("GET")

	r.Handle("/oauth2/connections/{id}", hydcon.NewContextAdapter(
		context.Background(),
		extractor,
		h.m.IsAuthenticated,
	).ThenFunc(h.Get)).Methods("GET")

	r.Handle("/oauth2/connections/{id}", hydcon.NewContextAdapter(
		context.Background(),
		extractor,
		h.m.IsAuthenticated,
	).ThenFunc(h.Delete)).Methods("DELETE")
}

func (h *Handler) Create(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	var conn DefaultConnection
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&conn); err != nil {
		HttpError(rw, err, http.StatusBadRequest)
		return
	}

	if v, err := govalidator.ValidateStruct(conn); !v {
		if err != nil {
			HttpError(rw, err, http.StatusBadRequest)
			return
		}
		HttpError(rw, errors.New("Payload did not validate."), http.StatusBadRequest)
		return
	}

	conn.ID = uuid.New()
	if err := h.s.Create(&conn); err != nil {
		HttpError(rw, err, http.StatusInternalServerError)
		return
	}

	WriteCreatedJSON(rw, "/oauth2/connections/"+conn.ID, &conn)
}

func (h *Handler) Find(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	subject, ok := mux.Vars(req)["subject"]
	if !ok {
		HttpError(rw, errors.New("No id given."), http.StatusBadRequest)
		return
	}

	h.m.IsAuthorized(connectionsPermission, "get", middleware.NewEnv(req).Owner(subject))(hydcon.ContextHandlerFunc(
		func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
			conns, err := h.s.FindAllByLocalSubject(subject)
			if err != nil {
				HttpError(rw, err, http.StatusNotFound)
				return
			}
			WriteJSON(rw, conns)
		},
	)).ServeHTTPContext(ctx, rw, req)
}

func (h *Handler) Get(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	id, ok := mux.Vars(req)["id"]
	if !ok {
		HttpError(rw, errors.New("No id given."), http.StatusBadRequest)
		return
	}

	conn, err := h.s.Get(id)
	if err != nil {
		WriteError(rw, err)
		return
	}

	h.m.IsAuthorized(permission(id), "get", middleware.NewEnv(req).Owner(conn.GetLocalSubject()))(hydcon.ContextHandlerFunc(
		func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
			WriteJSON(rw, conn)
		},
	)).ServeHTTPContext(ctx, rw, req)
}

func (h *Handler) Delete(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
	id, ok := mux.Vars(req)["id"]
	if !ok {
		http.Error(rw, "No id given.", http.StatusBadRequest)
		return
	}

	conn, err := h.s.Get(id)
	if err != nil {
		HttpError(rw, err, http.StatusNotFound)
		return
	}

	h.m.IsAuthorized(permission(id), "delete", middleware.NewEnv(req).Owner(conn.GetLocalSubject()))(hydcon.ContextHandlerFunc(
		func(ctx context.Context, rw http.ResponseWriter, req *http.Request) {
			if err := h.s.Delete(conn.GetID()); err != nil {
				HttpError(rw, err, http.StatusInternalServerError)
				return
			}
			rw.WriteHeader(http.StatusAccepted)
		},
	)).ServeHTTPContext(ctx, rw, req)
}
