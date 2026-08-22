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

func (h *providerHandler) AddProvider(w http.ResponseWriter, r *http.Request) Response {

	cfg, err := decode[model.ProviderConfig](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.providerSvc.AddProvider(cfg); err != nil {
		if errors.Is(err, types.ErrAlreadyExist) {
			return NewBadRequest("provider already exist")
		}
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func (h *providerHandler) DeleteProvider(w http.ResponseWriter, r *http.Request) Response {
	providerID := model.ProviderID(r.PathValue("name"))

	if err := h.providerSvc.DeleteProvider(providerID); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("provider is not exist")
		}
		return NewInternalError(err)
	}
	return NewResponse(http.StatusOK)
}

func (h *providerHandler) GetProvider(w http.ResponseWriter, r *http.Request) Response {
	providerID := model.ProviderID(r.PathValue("name"))

	cfg, err := h.providerSvc.GetProvider(providerID)
	if err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("provider is not exist")
		}
		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, cfg)
}

func (h *providerHandler) UpdateProvider(w http.ResponseWriter, r *http.Request) Response {

	providerID := model.ProviderID(r.PathValue("name"))

	patch, err := decode[model.ProviderConfigPatch](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.providerSvc.UpdateProvider(providerID, patch); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewNotFound("provider is not exist")
		}

		// TODO: validation error?

		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func (h *providerHandler) GetAll(w http.ResponseWriter, r *http.Request) Response {
	all, err := h.providerSvc.GetAll()
	if err != nil {
		return NewInternalError(err)
	}

	return NewJSONResponse(http.StatusOK, all)
}

func (h *providerHandler) SetModel(w http.ResponseWriter, r *http.Request) Response {

	providerID := model.ProviderID(r.PathValue("name"))
	encodedModelName := r.PathValue("model")

	modelNameData, err := base64.URLEncoding.DecodeString(encodedModelName)
	if err != nil {
		return NewBadRequest("model_name bad encoding")
	}

	modelCfg, err := decode[model.ModelConfig](r)
	if err != nil {
		return NewInvalidRequest(err)
	}

	if err := h.providerSvc.SetModel(providerID, string(modelNameData), modelCfg); err != nil {

		// TODO: validation errors in fact
		switch {
		case errors.Is(err, model.ErrEmptyModelName):
			return NewBadRequest("empty model name")
		case errors.Is(err, model.ErrUnsupportedAPI):
			return NewBadRequest("api is not supported")
		}

		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}

func (h *providerHandler) DeleteModel(w http.ResponseWriter, r *http.Request) Response {
	providerID := model.ProviderID(r.PathValue("name"))
	encodedModelName := r.PathValue("model")

	modelNameData, err := base64.URLEncoding.DecodeString(encodedModelName)
	if err != nil {
		return NewBadRequest("model_name bad encoding")
	}

	if err := h.providerSvc.DeleteModel(providerID, string(modelNameData)); err != nil {
		if errors.Is(err, types.ErrIsNotExist) {
			return NewBadRequest("is not exist")
		}
		return NewInternalError(err)
	}

	return NewResponse(http.StatusOK)
}
