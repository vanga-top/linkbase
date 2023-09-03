package generate

import "github.com/linkbase/middleware"

type UniqueID = middleware.UniqueID

type Generate interface {
	Gen(count uint32) (UniqueID, UniqueID, error)
	GenOne() (UniqueID, error)
}
