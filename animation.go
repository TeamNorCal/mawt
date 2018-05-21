package mawt

// This file contains the interfaces to the TeamNorCal animation suite
//
// This package will request sets of frames from the animation library that
// represent actions occuring within the portal and will play these back
// to the fadecandy server interface

import (
	"time"

	"github.com/TeamNorCal/animation"
	"github.com/TeamNorCal/mawt/model"
	"github.com/karlmutch/errors"
)

type statusSink struct {
	statusC chan *model.PortalStatus
}

var animPortal = animation.NewPortal()

func NewSink() (sink *statusSink) {
	return &statusSink{
		statusC: make(chan *model.PortalStatus),
	}
}

func (sink *statusSink) UpdateStatus(status *model.Status) (err errors.Error) {
	animPortal.UpdateStatus(model.StatusToAnimation(status))
	return nil
}

func (sink *statusSink) GetFrame(tm time.Time) (data []animation.ChannelData) {
	return animPortal.GetFrame(tm)
}
