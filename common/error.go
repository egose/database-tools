package common

import (
	"github.com/mongodb/mongo-tools/common/log"
)

type Block struct {
	Try     func()
	Catch   func(Exception)
	Finally func()
}

type Exception interface{}

func Throw(up Exception) {
	panic(up)
}

func (tcf Block) Do() {
	if tcf.Finally != nil || tcf.Catch != nil {
		defer func() {
			if r := recover(); r != nil {
				if tcf.Catch != nil {
					tcf.Catch(r)
				}
			}
			if tcf.Finally != nil {
				tcf.Finally()
			}
		}()
	}
	tcf.Try()
}

func HandleErrorToPanic(err error) {
	if err != nil {
		log.Logvf(log.Always, "Failed in Panic: %v", err.Error())
		panic("Execution stopped due to error")
	}
}
