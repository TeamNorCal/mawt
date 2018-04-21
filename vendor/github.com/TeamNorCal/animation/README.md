# Pixel Animation Support

This module provides support for generating animations of pixels and for
marshaling pixel data into a format that can easily be sent to a pixel
controller.

This consists of several distinct areas, summarized below.

## Logical-to-Physical Mapping

This area consists of the following tasks:

* Support a logical view of the pixel model, consisting of a set of 'universes',
  each of which is a linear array of pixels corresponding to some model element.
* Allow definition of a mapping from the logical 'universe' view of the model
  to physical pixel layout defined by a number of controller boards, pixel
  strands within the controller boards, and pixels within the strands. A logical
  universe can span multiple physical strands and occupy only portions of
  strands.
* Maintaining a buffer of pixel data, which can be updated by providing new
  frame data at the granularity of a universe, and retrieved at the granularity
  of a strand, to be sent to the pixel controller.

See `universe.go`.

## Effect Definition

Implementations of various animation effects, such as fades, keyframe-based
animations, 'walking' pixels, etc. These effects can be parameterized by color,
duration, etc. as appropriate for the effect in question. They then produce
frames of pixel data for a universe to realize that effect.

## Effect Sequencing

Effects are the building blocks of animations. Sitting on top of the effects
is a sequencing layer that orchestrates effects across universes in order to
achieve a particular sequence of animations. A change in portal state will
typically trigger an effect sequence.
