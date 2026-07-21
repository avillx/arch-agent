package api

import (
	"arch-agent/internal/model"
	"arch-agent/internal/types"
	"encoding/base64"
	"errors"
	"net/http"
)

// handler
type providerHandler struct {
	providerSvc *model.ProviderService
}

func (h *providerHandler) AddProvider(w http.ResponseWriter, r *http.Request) error {
	cfg, err := decodeValid[model.ProviderConfig](r)
	if err != nil {
		// already errStatus
		return err
	}

	if err := h.providerSvc.AddProvider(cfg); err != nil {
		if errors.Is(err, types.ErrAlreadyExist) {
			return badRequest("provider already exist")
		}
		return internal(err)
	}

	return respond(w, http.StatusCreated, message("success"))
}

func (h *providerHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")

	if err := h.providerSvc.DeleteProvider(model.ProviderID(name)); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("is not exist")
		}
		return err
	}

	return nil
}

func (h *providerHandler) GetProvider(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")

	cfg, err := h.providerSvc.GetProvider(model.ProviderID(name))
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("is not exist")
		}
		return err
	}

	return respond(w, http.StatusOK, cfg)
}

func (h *providerHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) error {

	name := r.PathValue("name")

	patch, err := decode[model.ProviderConfigPatch](r)
	if err != nil {
		return err
	}

	if err := h.providerSvc.UpdateProvider(model.ProviderID(name), patch); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("provider is not exist")
		}
		return err
	}

	return respond(w, http.StatusOK, message("success"))
}

func (h *providerHandler) GetAll(w http.ResponseWriter, r *http.Request) error {
	all, err := h.providerSvc.GetAll()
	if err != nil {
		return internal(err)
	}

	return respond(w, http.StatusOK, all)
}

func (h *providerHandler) SetModel(w http.ResponseWriter, r *http.Request) error {

	name := r.PathValue("name")
	encodedModelName := r.PathValue("model")

	modelNameData, err := base64.URLEncoding.DecodeString(encodedModelName)
	if err != nil {
		return badRequest("model_name bad encoding")
	}

	modelCfg, err := decode[model.ModelConfig](r)
	if err != nil {
		return err
	}

	if err := h.providerSvc.SetModel(model.ProviderID(name), string(modelNameData), modelCfg); err != nil {
		switch {
		case errors.Is(err, model.ErrEmptyModelName):
			return badRequest("empty model name")
		case errors.Is(err, model.ErrUnsupportedAPI):
			return badRequest("api is not supported")
		}

		return err
	}

	return respond(w, http.StatusOK, message("success"))
}

func (h *providerHandler) DeleteModel(w http.ResponseWriter, r *http.Request) error {
	name := r.PathValue("name")
	encodedModelName := r.PathValue("model")

	modelNameData, err := base64.URLEncoding.DecodeString(encodedModelName)
	if err != nil {
		return badRequest("model_name bad encoding")
	}

	if err := h.providerSvc.DeleteModel(model.ProviderID(name), string(modelNameData)); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return badRequest("is not exist")
		}
		return err
	}

	return respond(w, http.StatusOK, message("deleted"))
}
