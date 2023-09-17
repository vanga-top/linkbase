package generator

import "github.com/linkbase/middleware"

type UniqueID = middleware.UniqueID

type Generator interface {
	Gen(count uint32) (UniqueID, UniqueID, error)
	GenOne() (UniqueID, error)
}
