package handler

import (
	"errors"

	"github.com/m1k1o/neko/server/pkg/types"
	"github.com/m1k1o/neko/server/pkg/types/event"
	"github.com/m1k1o/neko/server/pkg/types/message"
)

func (h *MessageHandlerCtx) screenSet(session types.Session, payload *message.ScreenSize) error {
	if !session.Profile().IsAdmin {
		return errors.New("is not the admin")
	}

	size, err := h.desktop.SetScreenSize(payload.ScreenSize)
	if err != nil {
		return err
	}

	h.sessions.Broadcast(event.SCREEN_UPDATED, message.ScreenSizeUpdate{
		ID:         session.ID(),
		ScreenSize: size,
	})
	return nil
}
