package timerecord

import "github.com/linkbase/utils"

var groups = utils.NewConcurrentMap[string, *GroupChecker]()

type GroupChecker struct {
}