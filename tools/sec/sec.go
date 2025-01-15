package sec

import (
	"github.com/named-data/ndnd/std/utils"
)

func GetSecCmdTree() utils.CmdTree {
	return utils.CmdTree{
		Name: "sec",
		Help: "NDN Security Utilities",
		Sub: []*utils.CmdTree{{
			Name: "txt-from",
			Help: "Convert an NDN data to text representation",
			Fun:  txtFrom,
		}, {
			Name: "txt-parse",
			Help: "Parse a text representation of an NDN data",
			Fun:  txtParse,
		}},
	}
}
