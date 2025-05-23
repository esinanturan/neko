package room

import (
	"net/http"

	"github.com/go-chi/chi"

	"github.com/m1k1o/neko/server/pkg/auth"
	"github.com/m1k1o/neko/server/pkg/types/event"
	"github.com/m1k1o/neko/server/pkg/types/message"
	"github.com/m1k1o/neko/server/pkg/utils"
)

type ControlStatusPayload struct {
	HasHost bool   `json:"has_host"`
	HostId  string `json:"host_id,omitempty"`
}

type ControlTargetPayload struct {
	ID string `json:"id"`
}

func (h *RoomHandler) controlStatus(w http.ResponseWriter, r *http.Request) error {
	host, hasHost := h.sessions.GetHost()

	var hostId string
	if hasHost {
		hostId = host.ID()
	}

	return utils.HttpSuccess(w, ControlStatusPayload{
		HasHost: hasHost,
		HostId:  hostId,
	})
}

func (h *RoomHandler) controlRequest(w http.ResponseWriter, r *http.Request) error {
	session, _ := auth.GetSession(r)
	host, hasHost := h.sessions.GetHost()
	if hasHost {
		// TODO: Some throttling mechanism to prevent spamming.

		// let host know that someone wants to take control
		host.Send(
			event.CONTROL_REQUEST,
			message.SessionID{
				ID: session.ID(),
			})

		return utils.HttpError(http.StatusAccepted, "control request sent")
	}

	if h.sessions.Settings().LockedControls && !session.Profile().IsAdmin {
		return utils.HttpForbidden("controls are locked")
	}

	session.SetAsHost()

	return utils.HttpSuccess(w)
}

func (h *RoomHandler) controlRelease(w http.ResponseWriter, r *http.Request) error {
	session, _ := auth.GetSession(r)
	if !session.IsHost() {
		return utils.HttpUnprocessableEntity("session is not the host")
	}

	h.desktop.ResetKeys()
	session.ClearHost()

	return utils.HttpSuccess(w)
}

func (h *RoomHandler) controlTake(w http.ResponseWriter, r *http.Request) error {
	session, _ := auth.GetSession(r)
	session.SetAsHost()

	return utils.HttpSuccess(w)
}

func (h *RoomHandler) controlGive(w http.ResponseWriter, r *http.Request) error {
	session, _ := auth.GetSession(r)
	sessionId := chi.URLParam(r, "sessionId")

	target, ok := h.sessions.Get(sessionId)
	if !ok {
		return utils.HttpNotFound("target session was not found")
	}

	if !target.Profile().CanHost {
		return utils.HttpBadRequest("target session is not allowed to host")
	}

	target.SetAsHostBy(session)

	return utils.HttpSuccess(w)
}

func (h *RoomHandler) controlReset(w http.ResponseWriter, r *http.Request) error {
	session, _ := auth.GetSession(r)
	_, hasHost := h.sessions.GetHost()

	if hasHost {
		h.desktop.ResetKeys()
		session.ClearHost()
	}

	return utils.HttpSuccess(w)
}
