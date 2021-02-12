// Renderers disabled if noim is specified
//

//+build !im

package main

import (
	"errors"

	"github.com/brunotm/goseq/seqdiagram"
)

func PngRenderer(diagram *seqdiagram.Diagram, opts *seqdiagram.ImageOptions, target string) error {
	return errors.New("PNG renderer not available")
}
