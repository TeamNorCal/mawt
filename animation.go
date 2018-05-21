package mawt

// This file contains the interfaces to the TeamNorCal animation suite
//
// This package will request sets of frames from the animation library that
// represent actions occuring within the portal and will play these back
// to the fadecandy server interface

import (
	"time"

	"github.com/karlmutch/errors"

	"github.com/TeamNorCal/animation"
	animationModel "github.com/TeamNorCal/animation/model"
	"github.com/TeamNorCal/mawt/model"
)

type statusSink struct {
	statusC chan *model.PortalStatus
	portal  animationModel.Portal
}

func NewSink() (sink *statusSink) {
	return &statusSink{
		statusC: make(chan *model.PortalStatus),
		portal:  animation.NewPortal(),
	}
}

func (sink *statusSink) UpdateStatus(status *model.Status) (err errors.Error) {
	sink.portal.UpdateFromCanonicalStatus(status)
	return nil
}

func (sink *statusSink) GetFrame(tm time.Time) []animationModel.ChannelData {
	return sink.portal.GetFrame(tm)
}
